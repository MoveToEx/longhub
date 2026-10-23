import { useEffect, useState, type ReactNode } from "react";
import { ArrowDown, ArrowUp, ArrowUpDown, Columns3, RefreshCw, Search } from "lucide-react";
import { Button } from "@/shared/components/ui/button";
import { Input } from "@/shared/components/ui/input";
import { Pagination } from "@/shared/components/ui/pagination";
import { Spinner } from "@/shared/components/ui/spinner";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/shared/components/ui/dropdown-menu";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/shared/components/ui/select";
import { useDebouncedState } from "@/shared/hooks/use-debounced-state";
import useAdminQuery from "../hooks/use-admin-query";
import type { AdminQueryInput, AdminResource, SortDirection } from "../lib/types";

export type AdminColumn<T> = {
  id: string;
  label: string;
  sortBy?: string;
  hidden?: boolean;
  render: (row: T) => ReactNode;
};

type Props<T> = {
  resource: AdminResource;
  title: string;
  searchPlaceholder: string;
  query: string;
  columns: AdminColumn<T>[];
};

export default function AdminTable<T extends { id: string }>({
  resource,
  title,
  searchPlaceholder,
  query,
  columns,
}: Props<T>) {
  const label = title.toLowerCase();
  const search = useDebouncedState("", 300);
  const [offset, setOffset] = useState(0);
  const [limit, setLimit] = useState(25);
  const [sortBy, setSortBy] = useState("CREATED_AT");
  const [direction, setDirection] = useState<SortDirection>("DESC");
  const [status, setStatus] = useState<NonNullable<AdminQueryInput["status"]>>("ACTIVE");
  const statuses: Record<string, string> =
    resource === "images"
      ? { ACTIVE: "Active", DELETED: "Deleted", ALL: "All images" }
      : {
          ACTIVE: "Active",
          APPROVED: "Approved",
          REJECTED: "Rejected",
          REVOKED: "Revoked",
          ALL: "All requests",
        };
  const [visible, setVisible] = useState(() =>
    columns.filter((column) => !column.hidden).map((column) => column.id),
  );
  const input: AdminQueryInput = {
    offset,
    limit,
    sortBy,
    direction,
    search: search.debouncedState,
    ...(resource !== "users" ? { status } : {}),
  };
  const { data, error, isValidating, mutate, unauthorized } = useAdminQuery<T>(
    resource,
    query,
    input,
    !search.isPending,
  );
  const shownColumns = columns.filter((column) => visible.includes(column.id));
  const total = data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / limit));

  useEffect(() => {
    if (data && offset > 0 && offset >= data.total)
      setOffset(Math.max(0, Math.ceil(data.total / limit) - 1) * limit);
  }, [data, offset, limit]);

  return (
    <section className="min-w-0 space-y-5" aria-label={`${title} administration`}>
      <div className="flex flex-wrap items-center gap-2">
        <div className="relative min-w-48 flex-1">
          <Search
            aria-hidden="true"
            className="absolute left-3 top-2.5 size-4 text-muted-foreground"
          />
          <Input
            aria-label={`Search ${title.toLowerCase()}`}
            className="pl-9"
            placeholder={searchPlaceholder}
            value={search.instantState}
            maxLength={255}
            onChange={(event) => {
              search.setState(event.target.value);
              setOffset(0);
            }}
          />
        </div>
        {resource !== "users" && (
          <Select
            value={status}
            onValueChange={(value) => {
              if (value) {
                setStatus(value);
                setOffset(0);
              }
            }}
          >
            <SelectTrigger aria-label={resource === "images" ? "Image status" : "Request status"}>
              <SelectValue>{(value) => statuses[String(value)]}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              {Object.entries(statuses).map(([value, label]) => (
                <SelectItem key={value} value={value}>
                  {label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button variant="outline">
                <Columns3 /> Columns
              </Button>
            }
          />
          <DropdownMenuContent align="end" className="min-w-48">
            {columns.map((column) => (
              <DropdownMenuCheckboxItem
                key={column.id}
                checked={visible.includes(column.id)}
                disabled={visible.length === 1 && visible.includes(column.id)}
                onCheckedChange={(checked) =>
                  setVisible((current) =>
                    checked ? [...current, column.id] : current.filter((id) => id !== column.id),
                  )
                }
                closeOnClick={false}
              >
                {column.label}
              </DropdownMenuCheckboxItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
        <Button
          variant="outline"
          aria-label={`Refresh ${label}`}
          disabled={isValidating || search.isPending}
          onClick={() => void mutate()}
        >
          <RefreshCw className={isValidating ? "animate-spin" : ""} />
          <span className="hidden sm:inline">Refresh</span>
        </Button>
      </div>
      {error ? (
        <div role="alert" className="space-y-3 rounded-xl border p-6">
          <p className="font-medium">
            {unauthorized ? "Admin access is no longer available." : `Unable to load ${label}.`}
          </p>
          <p className="text-sm text-muted-foreground">
            {unauthorized
              ? "Your session may have expired or your permissions may have changed."
              : error instanceof Error
                ? error.message
                : "Please try again."}
          </p>
          {!unauthorized && (
            <Button variant="outline" onClick={() => void mutate()}>
              Retry
            </Button>
          )}
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border bg-card">
          <div className="w-full overflow-x-auto">
            <table
              className="w-full text-left text-sm"
              aria-label={`${title} table`}
              aria-busy={!data || isValidating}
            >
              <thead className="border-b bg-muted/40">
                <tr>
                  {shownColumns.map((column) => (
                    <th
                      key={column.id}
                      scope="col"
                      className="whitespace-nowrap px-4 py-3 font-medium"
                      aria-sort={
                        column.sortBy === sortBy
                          ? direction === "ASC"
                            ? "ascending"
                            : "descending"
                          : undefined
                      }
                    >
                      {column.sortBy ? (
                        <button
                          className="flex items-center gap-2 rounded-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
                          onClick={() => {
                            setDirection(
                              column.sortBy === sortBy && direction === "DESC" ? "ASC" : "DESC",
                            );
                            setSortBy(column.sortBy!);
                            setOffset(0);
                          }}
                        >
                          {column.label}
                          {column.sortBy === sortBy ? (
                            direction === "ASC" ? (
                              <ArrowUp className="size-3.5" />
                            ) : (
                              <ArrowDown className="size-3.5" />
                            )
                          ) : (
                            <ArrowUpDown className="size-3.5 text-muted-foreground" />
                          )}
                        </button>
                      ) : (
                        column.label
                      )}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {!data ? (
                  <tr>
                    <td colSpan={shownColumns.length} className="px-4 py-14 text-center">
                      <span role="status" className="inline-flex items-center gap-2">
                        <Spinner /> Loading {label}…
                      </span>
                    </td>
                  </tr>
                ) : data.items.length === 0 ? (
                  <tr>
                    <td
                      colSpan={shownColumns.length}
                      className="px-4 py-14 text-center text-muted-foreground"
                    >
                      No {label} match your filters.
                    </td>
                  </tr>
                ) : (
                  data.items.map((row) => (
                    <tr key={row.id} className="border-b last:border-b-0 hover:bg-muted/30">
                      {shownColumns.map((column) => (
                        <td key={column.id} className="px-4 py-3 align-top">
                          {column.render(row)}
                        </td>
                      ))}
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
          <div className="flex flex-wrap items-center justify-between gap-4 border-t px-4 py-3">
            <p className="text-sm text-muted-foreground" aria-live="polite">
              {data
                ? total === 0
                  ? "0 results"
                  : `${offset + 1}–${Math.min(offset + limit, total)} of ${total.toLocaleString()}`
                : "Loading results…"}
            </p>
            <div className="flex items-center gap-2 text-sm">
              <span>Rows per page</span>
              <Select
                value={limit}
                onValueChange={(value) => {
                  if (value) {
                    setLimit(value);
                    setOffset(0);
                  }
                }}
              >
                <SelectTrigger aria-label="Rows per page" size="sm">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {[25, 50, 100].map((size) => (
                    <SelectItem key={size} value={size}>
                      {size}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <Pagination
              className="w-auto"
              count={pageCount}
              page={Math.floor(offset / limit) + 1}
              onChange={(_, page) => setOffset((page - 1) * limit)}
              disabled={!data || isValidating}
            />
          </div>
        </div>
      )}
    </section>
  );
}
