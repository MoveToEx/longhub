import { beforeEach, expect, it, vi } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { AxiosError } from "axios";
import AdminUsersPage from "@/features/admin/pages/admin-users-page";
import type { AdminUser } from "@/features/admin/lib/types";

const mocks = vi.hoisted(() => ({
  auth: { data: { id: 1, permission: 8 }, session: "admin-session", isLoading: false, mutate: vi.fn() },
  post: vi.fn(),
}));
vi.mock("@/features/auth/hooks/use-auth", () => ({ default: () => mocks.auth }));
vi.mock("@/shared/lib/axios", () => ({ default: { post: mocks.post } }));

let managedUser: AdminUser;

function mount() {
  return render(
    <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0, revalidateOnFocus: false }}>
      <MemoryRouter><AdminUsersPage /></MemoryRouter>
    </SWRConfig>,
  );
}

async function openDialog() {
  await userEvent.click(await screen.findByRole("button", { name: "Manage target" }));
  return screen.getByRole("dialog", { name: "Manage user" });
}

async function transferPermission(dialog: HTMLElement, name: string, direction: "grant" | "revoke") {
  const list = within(dialog).getByRole("group", {
    name: direction === "grant" ? "Available permissions" : "Granted permissions",
  });
  await userEvent.click(within(list).getByRole("button", { name }));
  await userEvent.click(within(dialog).getByRole("button", {
    name: direction === "grant" ? "Grant selected permissions" : "Revoke selected permissions",
  }));
}

beforeEach(() => {
  managedUser = {
    id: "9007199254740993", username: "target", email: "old@example.com", permission: 3,
    createdAt: "2026-10-23T00:00:00Z", activeImageCount: 2, versionCount: 3,
  };
  mocks.auth.mutate.mockReset();
  mocks.auth.data = { id: 1, permission: 8 };
  mocks.post.mockReset();
  mocks.post.mockImplementation(async (_url, { query, variables }) => {
    if (query.includes("mutation UpdateUser")) {
      const mask = mocks.auth.data.permission & 16 ? 15 : 7;
      managedUser = { ...managedUser, ...variables.input, permission: (managedUser.permission & ~mask) | (variables.input.permission & mask) };
      return { data: { data: { updateUser: managedUser } } };
    }
    return { data: { data: { users: { items: [managedUser], total: 1 } } } };
  });
});

it.each([3, 11])("limits transfers to editable permissions for permission %s", async (permission) => {
  managedUser.permission = permission;
  mount();
  const dialog = await openDialog();
  expect(within(dialog).getByRole("button", { name: "Drag down to close drawer" })).toBeTruthy();
  expect((within(dialog).getByRole("textbox", { name: "Email" }) as HTMLInputElement).value).toBe("old@example.com");
  expect(within(dialog).queryByRole("button", { name: "Admin" })).toBeNull();
  expect(within(dialog).queryByRole("button", { name: "Root" })).toBeNull();
  await transferPermission(dialog, "Issue deletion", "grant");
  expect(within(within(dialog).getByRole("group", { name: "Granted permissions" })).getByRole("button", { name: "Issue deletion" })).toBeTruthy();
  expect(mocks.post.mock.calls.every(([, body]) => !body.query.includes("mutation"))).toBe(true);
  await userEvent.click(within(dialog).getByRole("button", { name: "Save changes" }));
  await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  expect(managedUser.permission).toBe(permission | 4);
  const mutation = mocks.post.mock.calls.find(([, body]) => body.query.includes("mutation UpdateUser"));
  expect(mutation?.[1].variables).toEqual({ id: managedUser.id, input: { email: "old@example.com", permission: 7 } });
});

it("saves email, password, and editable permissions and refreshes the table", async () => {
  mount();
  const dialog = await openDialog();
  const user = userEvent.setup();
  const email = within(dialog).getByRole("textbox", { name: "Email" });
  await user.clear(email);
  await user.type(email, "updated@example.com");
  await user.type(within(dialog).getByLabelText("New password"), "replacement-password");
  await transferPermission(dialog, "Create new image", "revoke");
  await user.click(within(dialog).getByRole("button", { name: "Save changes" }));
  await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  expect(await screen.findByText("updated@example.com")).toBeTruthy();
  expect(mocks.post).toHaveBeenCalledWith(
    "/admin/graphql",
    expect.objectContaining({ variables: { id: managedUser.id, input: { email: "updated@example.com", password: "replacement-password", permission: 2 } } }),
    { headers: { Authorization: "Session admin-session" } },
  );
  await openDialog();
  expect((screen.getByLabelText("New password") as HTMLInputElement).value).toBe("");
});

it("discards draft changes when cancelled and restores the current user on reopening", async () => {
  mount();
  const dialog = await openDialog();
  const user = userEvent.setup();
  await user.clear(within(dialog).getByRole("textbox", { name: "Email" }));
  await user.type(within(dialog).getByRole("textbox", { name: "Email" }), "draft@example.com");
  await user.type(within(dialog).getByLabelText("New password"), "draft-password");
  await transferPermission(dialog, "Issue deletion", "grant");
  await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
  await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  await openDialog();
  expect((screen.getByRole("textbox", { name: "Email" }) as HTMLInputElement).value).toBe("old@example.com");
  expect((screen.getByLabelText("New password") as HTMLInputElement).value).toBe("");
  expect(within(screen.getByRole("group", { name: "Available permissions" })).getByRole("button", { name: "Issue deletion" })).toBeTruthy();
  expect(mocks.post.mock.calls.every(([, body]) => !body.query.includes("mutation"))).toBe(true);
});

it("keeps failed changes open for retry and rechecks identity on access denial", async () => {
  mount();
  const dialog = await openDialog();
  mocks.post.mockResolvedValueOnce({ data: { errors: [{ message: "Unable to update user" }] } });
  await userEvent.click(within(dialog).getByRole("button", { name: "Save changes" }));
  expect(await screen.findByRole("alert")).toHaveProperty("textContent", "Unable to update user");
  mocks.post.mockRejectedValueOnce(new AxiosError("Admin permission required", undefined, undefined, undefined, { status: 403 } as never));
  await userEvent.click(within(dialog).getByRole("button", { name: "Save changes" }));
  await waitFor(() => expect(mocks.auth.mutate).toHaveBeenCalledOnce());
  expect(screen.getByRole("dialog")).toBeTruthy();
  await userEvent.click(within(dialog).getByRole("button", { name: "Save changes" }));
  await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
});

it("validates password length in UTF-8 bytes before sending a mutation", async () => {
  mount();
  const dialog = await openDialog();
  await userEvent.type(within(dialog).getByLabelText("New password"), "密".repeat(25));
  await userEvent.click(within(dialog).getByRole("button", { name: "Save changes" }));
  expect(await screen.findByRole("alert")).toHaveProperty("textContent", "Password must be at most 72 bytes.");
  expect(mocks.post.mock.calls.every(([, body]) => !body.query.includes("mutation"))).toBe(true);
});

it.each([16, 24])("lets Root users grant and revoke A with actor permission %s", async (permission) => {
  mocks.auth.data.permission = permission;
  mount();
  const dialog = await openDialog();
  expect(within(dialog).queryByRole("button", { name: "Root" })).toBeNull();
  await transferPermission(dialog, "Admin", "grant");
  await userEvent.click(within(dialog).getByRole("button", { name: "Save changes" }));
  await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  expect(managedUser.permission).toBe(11);
  expect(mocks.post).toHaveBeenCalledWith(
    "/admin/graphql",
    expect.objectContaining({ variables: { id: managedUser.id, input: { email: "old@example.com", permission: 11 } } }),
    expect.anything(),
  );
  const reopened = await openDialog();
  await transferPermission(reopened, "Admin", "revoke");
  await userEvent.click(within(reopened).getByRole("button", { name: "Save changes" }));
  await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  expect(managedUser.permission).toBe(3);
});

it("prevents ordinary admins from editing Root accounts", async () => {
  managedUser.permission = 24;
  mount();
  const manage = await screen.findByRole("button", { name: "Manage target" });
  expect(manage.hasAttribute("disabled")).toBe(true);
  await userEvent.click(manage);
  expect(screen.queryByRole("dialog")).toBeNull();
});

it("preserves R when a Root user edits another Root account", async () => {
  mocks.auth.data.permission = 16;
  managedUser.permission = 24;
  mount();
  const dialog = await openDialog();
  expect(within(dialog).queryByRole("button", { name: "Root" })).toBeNull();
  await transferPermission(dialog, "Admin", "revoke");
  await userEvent.click(within(dialog).getByRole("button", { name: "Save changes" }));
  await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  expect(managedUser.permission).toBe(16);
});
