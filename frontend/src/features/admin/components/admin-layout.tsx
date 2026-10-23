import { useEffect, useRef } from 'react';
import { Link, NavLink, Outlet } from 'react-router';
import { useSWRConfig } from 'swr';
import { Images, Shield, Users } from 'lucide-react';
import useAuth from '@/features/auth/hooks/use-auth';
import { hasAdminPermission } from '@/shared/lib/const';
import { Spinner } from '@/shared/components/ui/spinner';
import { cn } from '@/shared/lib/utils';

export default function AdminLayout() {
  const { data, session, isLoading } = useAuth();
  const { mutate } = useSWRConfig();
  const allowed = Boolean(session) && hasAdminPermission(data?.permission);
  const previousSession = useRef(session);

  useEffect(() => {
    const previous = previousSession.current;
    previousSession.current = session;
    if (previous && (previous !== session || !allowed)) {
      void mutate(
        (key) => Array.isArray(key) && key[0] === 'admin' && key[1] === previous,
        undefined,
        { revalidate: false },
      );
    }
  }, [session, allowed, mutate]);

  if (isLoading)
    return (
      <div role="status" className="flex items-center gap-2 py-12">
        <Spinner /> Checking admin access…
      </div>
    );
  if (!allowed)
    return (
      <div role="alert" className="mx-auto max-w-lg space-y-3 rounded-xl border p-8">
        <Shield className="size-8 text-muted-foreground" />
        <h1 className="text-xl font-semibold">Admin access required</h1>
        <p className="text-muted-foreground">
          {data
            ? 'Your account does not have Admin permission.'
            : 'Log in with an account that has Admin permission to open this panel.'}
        </p>
        <Link to="/" className="inline-block underline underline-offset-4">
          Return home
        </Link>
      </div>
    );

  return (
    <div className="min-w-0 space-y-6">
      <header className="space-y-4">
        <nav aria-label="Admin pages" className="flex gap-1 border-b">
          {[
            { to: '/admin/images', label: 'Images', icon: Images },
            { to: '/admin/users', label: 'Users', icon: Users },
          ].map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-2 border-b-2 px-4 py-3 text-sm',
                  isActive
                    ? 'border-primary font-medium text-foreground'
                    : 'border-transparent text-muted-foreground hover:text-foreground',
                )
              }
            >
              <item.icon className="size-4" /> {item.label}
            </NavLink>
          ))}
        </nav>
      </header>
      <Outlet key={session} />
    </div>
  );
}
