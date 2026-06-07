import { Bookmark, ChevronUp, CircleUserRound, ExternalLink, FileText, Home, LogIn, LogOut, Settings, SquareMousePointer, User2 } from "lucide-react";
import { Sidebar, SidebarContent, SidebarFooter, SidebarGroup, SidebarGroupContent, SidebarMenu, SidebarMenuButton, SidebarMenuItem } from "@/shared/components/ui/sidebar";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/shared/components/ui/dropdown-menu";
import useAuth from "@/features/auth/hooks/use-auth";
import { Spinner } from "@/shared/components/ui/spinner";
import { Dialog as BaseDialog } from '@base-ui/react';
import LoginDialog from "@/features/auth/components/login-dialog";
import { Link, useLocation, useNavigate } from "react-router";
import { type FC } from "react";

const items = [
  {
    title: "Home",
    url: "/",
    icon: Home,
  },
  {
    title: "Browse",
    url: '/image',
    icon: SquareMousePointer,
  },
  // {
  //   title: "Search",
  //   url: "/search",
  //   icon: Search,
  // },
  {
    title: "Favorite",
    url: "/favorite",
    icon: Bookmark
  },
  {
    title: "Document",
    url: "https://doc.longhub.top/",
    icon: FileText
  },
];

const loginHandle = BaseDialog.createHandle<void>();

function AccountMenu() {
  const { data, error, isLoading, reset, mutate } = useAuth();
  const navigate = useNavigate();

  if (isLoading) {
    return (
      <>
        <LoginDialog handle={loginHandle} />
        <BaseDialog.Trigger handle={loginHandle} disabled render={
          <SidebarMenuButton className='h-10'>
            <Spinner /> Loading
          </SidebarMenuButton>
        } />
      </>
    )
  }

  if (error || !data) {
    return (
      <>
        <LoginDialog handle={loginHandle} />
        <BaseDialog.Trigger handle={loginHandle} render={
          <SidebarMenuButton className='h-10'>
            <LogIn /> Login
          </SidebarMenuButton>
        } />
      </>
    )
  }

  return (
    <>
      <LoginDialog handle={loginHandle} />
      <DropdownMenu>
        <DropdownMenuTrigger render={
          <SidebarMenuButton className='h-10'>
            <User2 /> {data.username}
            <ChevronUp className='ml-auto' />
          </SidebarMenuButton>
        } />
        <DropdownMenuContent side='top' className="w-[--radix-popper-anchor-width]">
          <DropdownMenuItem className='cursor-pointer' onClick={() => {
            navigate('/user/' + data.id)
          }}>
            <CircleUserRound /> Profile
          </DropdownMenuItem>
          <DropdownMenuItem className='cursor-pointer' onClick={() => {
            navigate('/user/settings')
          }}>
            <Settings /> Settings
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={() => {
            reset();
            mutate();
          }}>
            <LogOut />
            Logout
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </>
  )
}

function SidebarItem({ url, title, Icon }: {
  url: string,
  title: string,
  Icon: FC
}) {
  const { pathname } = useLocation();
  const isExternal = /^https?:\/\//.test(url);
  const isActive = !isExternal && (pathname === url || url !== '/' && pathname.startsWith(`${url}/`));

  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        isActive={isActive}
        render={
          isExternal ? (
            <a href={url} target='_blank' rel='noreferrer'>
              <Icon />
              <span>{title}</span>
              <ExternalLink className='text-muted-foreground max-w-3 max-h-3' />
            </a>
          ) : (
            <Link to={url}>
              <Icon />
              <span>{title}</span>
            </Link>
          )
        } />
    </SidebarMenuItem>
  )
}

export default function AppSidebar() {
  return (
    <Sidebar collapsible="icon">
      <SidebarContent>
        <SidebarGroup className='overflow-x-hidden'>
          <SidebarGroupContent>
            {items.map(({ url, icon, title }) => (
              <SidebarItem url={url} title={title} Icon={icon} key={title} />
            ))}
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <AccountMenu />
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  )
}
