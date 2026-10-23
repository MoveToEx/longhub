import { Link } from 'react-router';
import { formatPermissions } from '@/shared/lib/const';
import AdminTable, { type AdminColumn } from '../components/admin-table';
import { USER_QUERY, type AdminUser } from '../lib/types';

const columns: AdminColumn<AdminUser>[] = [
  { id: 'id', label: 'ID', sortBy: 'ID', render: (row) => row.id },
  {
    id: 'username',
    label: 'Username',
    sortBy: 'USERNAME',
    render: (row) => (
      <Link className="whitespace-nowrap underline underline-offset-4" to={`/user/${row.id}`}>
        {row.username}
      </Link>
    ),
  },
  { id: 'email', label: 'Email', sortBy: 'EMAIL', render: (row) => row.email },
  {
    id: 'permission',
    label: 'Permissions',
    sortBy: 'PERMISSION',
    render: (row) => (
      <code className="whitespace-nowrap font-mono">{formatPermissions(row.permission)}</code>
    ),
  },
  {
    id: 'createdAt',
    label: 'Created',
    sortBy: 'CREATED_AT',
    render: (row) => (
      <time dateTime={row.createdAt} className="whitespace-nowrap">
        {new Date(row.createdAt).toLocaleString()}
      </time>
    ),
  },
  { id: 'images', label: 'Active images', render: (row) => row.activeImageCount.toLocaleString() },
  { id: 'versions', label: 'Versions', render: (row) => row.versionCount.toLocaleString() },
];

export default function AdminUsersPage() {
  return (
    <AdminTable
      resource="users"
      title="Users"
      searchPlaceholder="Search by ID, username, or email…"
      query={USER_QUERY}
      columns={columns}
    />
  );
}
