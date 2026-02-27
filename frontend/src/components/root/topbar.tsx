import { SidebarTrigger } from "@/components/ui/sidebar";
import UploadDialog from "../dialogs/upload";
import QuickSearch from "../dialogs/quick-search";
import { ModeToggle } from "../mode-toggle";

export default function AppTopBar() {
  return (
    <div className='w-full border-b border-gray-250 h-12 flex flex-row items-center px-4 gap-2'>
      <SidebarTrigger />

      <a className='text-lg flex items-center ml-2' href='/#/'>
        L<img src='/o.png' alt='O' className='inline h-6' />NG&nbsp;<span className='hidden sm:block'>Hub</span>
      </a>

      <div className='flex-1' />

      <div>
        <ModeToggle />
      </div>

      <div>
        <QuickSearch />
      </div>

      <div>
        <UploadDialog />
      </div>
    </div>
  )
}