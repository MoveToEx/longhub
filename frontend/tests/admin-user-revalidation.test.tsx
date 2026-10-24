import { beforeEach, expect, it, vi } from "vitest";
import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { SWRConfig } from "swr";
import AdminLayout from "@/features/admin/components/admin-layout";
import AdminUsersPage from "@/features/admin/pages/admin-users-page";
import type { AdminUser } from "@/features/admin/lib/types";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));
vi.mock("@/shared/lib/axios", () => ({ default: { get: mocks.get, post: mocks.post } }));

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => { resolve = done; });
  return { promise, resolve };
}

const actor = {
  id: 1, username: "admin", email: "admin@example.com", permission: 8,
  createdAt: "2026-10-23T00:00:00Z", preference: {}, authorizedVia: "Session",
};
const record: AdminUser = {
  id: "7", username: "target", email: "old@example.com", permission: 3,
  createdAt: actor.createdAt, activeImageCount: 2, versionCount: 3,
};

beforeEach(() => {
  mocks.get.mockReset();
  mocks.post.mockReset();
  localStorage.setItem("nmsl-session", JSON.stringify("admin-session"));
});

it.each(["auth first", "users first"])("shows saved user data after revalidation resolves %s without another refresh", async (order) => {
  const authRefresh = deferred<{ data: { data: typeof actor } }>();
  const usersRefresh = deferred<{ data: { data: { users: { items: AdminUser[]; total: number } } } }>();
  const updated = { ...record, email: "updated@example.com" };
  let userQueries = 0;
  mocks.get.mockResolvedValueOnce({ data: { data: actor } }).mockImplementation(() => authRefresh.promise);
  mocks.post.mockImplementation(async (_url, { query }) => {
    if (query.includes("mutation UpdateUser")) return { data: { data: { updateUser: updated } } };
    userQueries++;
    return userQueries === 1
      ? { data: { data: { users: { items: [record], total: 1 } } } }
      : usersRefresh.promise;
  });
  const cache = new Map();
  render(
    <SWRConfig value={{ provider: () => cache, revalidateOnFocus: false }}>
      <MemoryRouter initialEntries={["/admin/users"]}>
        <Routes>
          <Route path="/admin" element={<AdminLayout />}>
            <Route path="users" element={<AdminUsersPage />} />
          </Route>
        </Routes>
      </MemoryRouter>
    </SWRConfig>,
  );
  const user = userEvent.setup();
  await user.click(await screen.findByRole("button", { name: "Manage target" }));
  const dialog = screen.getByRole("dialog");
  const email = within(dialog).getByRole("textbox", { name: "Email" });
  await user.clear(email);
  await user.type(email, updated.email);
  await user.click(within(dialog).getByRole("button", { name: "Save changes" }));
  await waitFor(() => {
    expect(mocks.get).toHaveBeenCalledTimes(2);
    expect(userQueries).toBe(2);
  });
  expect(screen.getByText(record.email)).toBeTruthy();
  expect(screen.queryByText("Checking admin access…")).toBeNull();
  const resolveAuth = () => act(async () => {
    authRefresh.resolve({ data: { data: actor } });
    await authRefresh.promise;
  });
  const resolveUsers = () => act(async () => {
    usersRefresh.resolve({ data: { data: { users: { items: [updated], total: 1 } } } });
    await usersRefresh.promise;
  });
  if (order === "auth first") {
    await resolveAuth();
    await resolveUsers();
  } else {
    await resolveUsers();
    await resolveAuth();
  }
  expect(await screen.findByText(updated.email)).toBeTruthy();
  expect(screen.queryByText("Loading users…")).toBeNull();
  expect(screen.queryByText("Checking admin access…")).toBeNull();
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(userQueries).toBe(2);
});
