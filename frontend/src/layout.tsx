import { Outlet } from "react-router";
import { SidebarProvider } from "@/components/ui/sidebar";
import AppSidebar from "@/components/root/sidebar";
import AppTopBar from "@/components/root/topbar";
import { Toaster } from "@/components/ui/sonner";
import { ThemeProvider } from "@/components/root/theme-provider";
import { ScrollArea } from "./components/ui/scroll-area";


export default function Layout() {
  return (
    <ThemeProvider defaultTheme='light' storageKey='vite-ui-theme'>
      <SidebarProvider>
        <div className='h-screen w-screen flex flex-row'>
          <Toaster position='top-center' />
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