import { beforeEach, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { AxiosError } from "axios";
import DeletionRequests from "@/features/images/components/deletion-requests";
import AdminDeletionsPage, { DeletionActions } from "@/features/admin/pages/admin-deletions-page";
import { invalidateDeletionCaches } from "@/features/images/lib/deletion-cache";
import type { AdminDeletion } from "@/features/admin/lib/types";

const mocks = vi.hoisted(() => ({
  auth: { data: { permission: 4 }, isLoading: false, session: "session", mutate: vi.fn() },
  get: vi.fn(),
  post: vi.fn(),
}));
vi.mock("@/features/auth/hooks/use-auth", () => ({ default: () => mocks.auth }));
vi.mock("@/shared/lib/axios", () => ({ default: { get: mocks.get, post: mocks.post } }));
const stamp = "2026-10-23T00:00:00Z";
const request = {
  id: 2,
  imageId: 7,
  reason: "duplicate original image",
  status: "active",
  requester: { id: 3, username: "reporter" },
  createdAt: stamp,
  processedAt: null,
};
const row: AdminDeletion = {
  ...request,
  id: "2",
  imageId: "7",
  requester: { id: "3", username: "reporter" },
  status: "ACTIVE",
  moderator: null,
  imageDeletedAt: null,
  activeDeletionId: null,
};

function mount(node: React.ReactNode) {
  const cache = new Map();
  return render(
    <SWRConfig value={{ provider: () => cache, dedupingInterval: 0, revalidateOnFocus: false }}>
      <MemoryRouter initialEntries={["/admin/deletions"]}>{node}</MemoryRouter>
    </SWRConfig>,
  );
}
beforeEach(() => {
  mocks.auth.data = { permission: 4 };
  mocks.auth.session = "session";
  mocks.auth.isLoading = false;
  mocks.auth.mutate.mockReset();
  mocks.get.mockReset();
  mocks.post.mockReset();
  mocks.get.mockResolvedValue({ data: { data: { items: [request], total: 1 } } });
  mocks.post.mockImplementation(async (_url, { query }) => ({
    data: {
      data:
        !query || query.includes("mutation")
          ? { moderateDeletionRequest: { id: "2", status: "APPROVED" } }
          : { deletionRequests: { items: [row], total: 60 } },
    },
  }));
});

it("keeps public history collapsed until opened and shows full history with pagination", async () => {
  mocks.auth.session = "";
  mocks.auth.data = { permission: 0 };
  mocks.get.mockResolvedValue({
    data: { data: { items: [{ ...request, status: "revoked", processedAt: stamp }], total: 12 } },
  });
  mount(<DeletionRequests id={7} deleted />);
  expect(mocks.get).not.toHaveBeenCalled();
  const user = userEvent.setup();
  await user.click(screen.getByRole("button", { name: "Deletion requests" }));
  expect(await screen.findByText(request.reason)).toBeTruthy();
  expect(screen.getByText("revoked")).toBeTruthy();
  expect(screen.getByRole("link", { name: "reporter" }).getAttribute("href")).toBe("/user/3");
  expect(screen.queryByRole("button", { name: "Request deletion" })).toBeNull();
  await user.click(screen.getByRole("button", { name: "Go to page 2" }));
  await waitFor(() => expect(mocks.get.mock.lastCall?.[1].params.offset).toBe(10));
});

it.each([8, 0])("does not allow submission without D (permission %s)", async (permission) => {
  mocks.auth.data = { permission };
  mount(<DeletionRequests id={7} deleted={false} />);
  await userEvent.click(screen.getByRole("button", { name: "Deletion requests" }));
  await screen.findByText(request.reason);
  expect(screen.queryByRole("button", { name: "Request deletion" })).toBeNull();
});

it("offers editable reason templates and submits a trimmed reason", async () => {
  mount(<DeletionRequests id={7} deleted={false} />);
  const user = userEvent.setup();
  await user.click(screen.getByRole("button", { name: "Deletion requests" }));
  await user.click(await screen.findByRole("button", { name: "Request deletion" }));
  await user.click(screen.getByRole("combobox", { name: "Reason template" }));
  expect(await screen.findByRole("option", { name: "Site-guideline violation" })).toBeTruthy();
  expect(screen.getByRole("option", { name: "Incorrect or misleading content" })).toBeTruthy();
  await user.click(screen.getByRole("option", { name: "Duplicate image" }));
  const reason = screen.getByRole("textbox", { name: "Reason" });
  expect((reason as HTMLTextAreaElement).value).toBe("This image is a duplicate of: ");
  await user.type(reason, "https://example.com/image/1  ");
  await user.click(screen.getByRole("button", { name: "Submit request" }));
  await waitFor(() =>
    expect(mocks.post).toHaveBeenCalledWith(
      "/image/7/deletion",
      { reason: "This image is a duplicate of: https://example.com/image/1" },
      expect.anything(),
    ),
  );
  await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
});

it("shows submission errors, revalidates identity on denial, and allows retry", async () => {
  mocks.post.mockRejectedValueOnce(
    new AxiosError("D permission revoked", undefined, undefined, undefined, {
      status: 403,
    } as never),
  );
  mount(<DeletionRequests id={7} deleted={false} />);
  const user = userEvent.setup();
  await user.click(screen.getByRole("button", { name: "Deletion requests" }));
  await user.click(await screen.findByRole("button", { name: "Request deletion" }));
  expect(screen.getByRole("button", { name: "Submit request" }).hasAttribute("disabled")).toBe(
    true,
  );
  await user.type(screen.getByRole("textbox", { name: "Reason" }), "reason");
  await user.click(screen.getByRole("button", { name: "Submit request" }));
  expect(await screen.findByRole("alert")).toBeTruthy();
  await waitFor(() => expect(mocks.auth.mutate).toHaveBeenCalled());
  await waitFor(() =>
    expect(screen.getByRole("button", { name: "Submit request" }).hasAttribute("disabled")).toBe(
      false,
    ),
  );
  await user.click(screen.getByRole("button", { name: "Submit request" }));
  await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  expect(mocks.post).toHaveBeenCalledTimes(2);
  expect(mocks.post).toHaveBeenLastCalledWith(
    "/image/7/deletion",
    { reason: "reason" },
    { headers: { Authorization: "Session session" } },
  );
});

it("defaults moderation to active and resets pagination when filtering by status", async () => {
  mocks.auth.data = { permission: 8 };
  mount(<AdminDeletionsPage />);
  const user = userEvent.setup();
  await screen.findByText(request.reason);
  expect(mocks.post.mock.lastCall?.[1].variables.input.status).toBe("ACTIVE");
  expect(screen.getByRole("link", { name: "#7" }).getAttribute("href")).toBe("/image/7");
  await user.click(screen.getByRole("button", { name: "Go to page 2" }));
  await waitFor(() => expect(mocks.post.mock.lastCall?.[1].variables.input.offset).toBe(25));
  await user.click(screen.getByRole("combobox", { name: "Request status" }));
  await user.click(await screen.findByRole("option", { name: "Approved" }));
  await waitFor(() =>
    expect(mocks.post.mock.lastCall?.[1].variables.input).toMatchObject({
      offset: 0,
      status: "APPROVED",
    }),
  );
});

it.each([
  ["ACTIVE", ["Accept", "Reject"]],
  ["APPROVED", ["Revoke"]],
  ["REJECTED", ["Reopen", "Accept"]],
  ["REVOKED", ["Reopen", "Accept"]],
] as const)("provides the appropriate actions for %s", (status, actions) => {
  mount(<DeletionActions row={{ ...row, status }} />);
  expect(screen.getAllByRole("button").map((button) => button.textContent)).toEqual(actions);
});

it("explains blocked acceptance while another request controls deletion", () => {
  mount(<DeletionActions row={{ ...row, imageDeletedAt: stamp, activeDeletionId: "99" }} />);
  expect(screen.getByRole("button", { name: "Accept" }).hasAttribute("disabled")).toBe(true);
  expect(screen.getByText(/Request #99 currently controls deletion/)).toBeTruthy();
  expect(screen.getByRole("button", { name: "Reject" }).hasAttribute("disabled")).toBe(false);
});

it("sends expected status, disables pending actions, and refreshes after conflict", async () => {
  mocks.auth.data = { permission: 8 };
  let finish!: (value: unknown) => void;
  mocks.post.mockImplementation(async (_url, { query }) =>
    query.includes("mutation")
      ? new Promise((resolve) => {
          finish = resolve;
        })
      : { data: { data: { deletionRequests: { items: [row], total: 1 } } } },
  );
  mount(<AdminDeletionsPage />);
  await screen.findByText(request.reason);
  await userEvent.click(screen.getByRole("button", { name: "Accept" }));
  expect(mocks.post.mock.lastCall?.[1].variables).toEqual({
    id: "2",
    action: "ACCEPT",
    expectedStatus: "ACTIVE",
  });
  expect(screen.getByRole("button", { name: "Reject" }).hasAttribute("disabled")).toBe(true);
  const calls = mocks.post.mock.calls.length;
  finish({ data: { errors: [{ message: "State changed; refresh and try again" }] } });
  await waitFor(() => expect(mocks.post.mock.calls.length).toBeGreaterThan(calls));
  await screen.findByText(request.reason);
});

it("invalidates tagged public data, favorites, counts, and both search array keys", async () => {
  const mutate = vi.fn().mockResolvedValue([]);
  await invalidateDeletionCaches(mutate);
  const predicate = mutate.mock.calls[0][0];
  for (const key of [
    ["admin", "session", "images"],
    ["/image/search", {}, 0, 25],
    ["/image/quick-search", []],
    { tags: ["user"] },
    { tags: ["image"] },
    { tags: ["favorite"], immutable: true },
  ])
    expect(predicate(key)).toBe(true);
  expect(predicate({ tags: ["auth", "user"] })).toBe(false);
  expect(mutate.mock.calls[0].slice(1)).toEqual([undefined, { revalidate: true }]);
});
