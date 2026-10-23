import { Link } from "react-router";
import AdminTable, { type AdminColumn } from "../components/admin-table";
import { IMAGE_QUERY, type AdminImage } from "../lib/types";

const date = (value: string) => (
  <time dateTime={value} className="whitespace-nowrap">
    {new Date(value).toLocaleString()}
  </time>
);
const columns: AdminColumn<AdminImage>[] = [
  {
    id: "id",
    label: "ID",
    sortBy: "ID",
    render: (row) => (
      <Link className="underline underline-offset-4" to={`/image/${row.id}`}>
        {row.id}
      </Link>
    ),
  },
  {
    id: "preview",
    label: "Preview",
    render: (row) => (
      <a href={row.imageUrl} target="_blank" rel="noreferrer">
        <img
          src={row.imageUrl}
          alt={`Image ${row.id}`}
          loading="lazy"
          className="size-16 rounded-md border bg-muted object-contain"
        />
      </a>
    ),
  },
  {
    id: "text",
    label: "Text",
    render: (row) => (
      <p className="min-w-48 max-w-80 whitespace-pre-wrap break-words">
        {row.currentVersion?.text || (
          <span className="text-muted-foreground">
            {row.currentVersion ? "No text" : "No current version"}
          </span>
        )}
      </p>
    ),
  },
  {
    id: "tags",
    label: "Tags",
    hidden: true,
    render: (row) => (
      <div className="flex min-w-32 max-w-64 flex-wrap gap-1">
        {row.currentVersion?.tags.map((tag) => (
          <span key={tag} className="rounded border px-1.5 py-0.5 text-xs">
            {tag}
          </span>
        ))}
        {!row.currentVersion?.tags.length && <span className="text-muted-foreground">—</span>}
      </div>
    ),
  },
  { id: "rating", label: "Rating", render: (row) => row.currentVersion?.rating ?? "—" },
  {
    id: "uploader",
    label: "Uploader",
    render: (row) => (
      <Link
        className="whitespace-nowrap underline underline-offset-4"
        to={`/user/${row.uploader.id}`}
      >
        {row.uploader.username} <span className="text-muted-foreground">#{row.uploader.id}</span>
      </Link>
    ),
  },
  {
    id: "status",
    label: "Status",
    render: (row) => (
      <span className="whitespace-nowrap rounded-full border px-2 py-1 text-xs">
        {row.deletedAt ? "Deleted" : "Active"}
      </span>
    ),
  },
  { id: "version", label: "Version", render: (row) => row.currentVersion?.version ?? "—" },
  { id: "createdAt", label: "Created", sortBy: "CREATED_AT", render: (row) => date(row.createdAt) },
  {
    id: "updatedAt",
    label: "Updated",
    sortBy: "UPDATED_AT",
    hidden: true,
    render: (row) => date(row.updatedAt),
  },
  {
    id: "deletedAt",
    label: "Deleted at",
    hidden: true,
    render: (row) => (row.deletedAt ? date(row.deletedAt) : "—"),
  },
  {
    id: "imageKey",
    label: "Storage key",
    hidden: true,
    render: (row) => <code className="block max-w-64 break-all text-xs">{row.imageKey}</code>,
  },
];

export default function AdminImagesPage() {
  return (
    <AdminTable
      resource="images"
      title="Images"
      searchPlaceholder="Search by ID, text, uploader, or tag…"
      query={IMAGE_QUERY}
      columns={columns}
    />
  );
}
