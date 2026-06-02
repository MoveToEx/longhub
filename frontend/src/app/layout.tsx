import { Outlet } from "react-router";
import { SidebarProvider } from "@/shared/components/ui/sidebar";
import AppSidebar from "@/app/components/sidebar";
import AppTopBar from "@/app/components/topbar";
import { Toaster } from "@/shared/components/ui/sonner";
import { ThemeProvider } from "@/app/components/theme-provider";
import { ScrollArea } from "@/shared/components/ui/scroll-area";
import EditShortcutDialog from "@/features/images/components/edit-shortcut-dialog";


export default function Layout() {
  return (
    <ThemeProvider defaultTheme='light' storageKey='vite-ui-theme'>
      <SidebarProvider>
        <div className='h-screen w-screen flex flex-row'>
          <Toaster position='top-center' />
          <EditShortcutDialog />
          <AppSidebar />
          <div className='w-full h-screen flex flex-col'>
            <AppTopBar />
            <ScrollArea className='flex-1 w-full overflow-auto'>
              <div className='w-full px-2 lg:px-16 md:px-8 py-4 md:py-6 lg:py-8'>
                <Outlet />
              </div>
            </ScrollArea>
          </div>
        </div>
      </SidebarProvider>
    </ThemeProvider>
  )
}
