import { SidebarTrigger } from "@/shared/components/ui/sidebar";
import UploadDialog from "@/features/images/components/upload-dialog";
import QuickSearch from "@/features/search/components/quick-search";
import { ModeToggle } from "@/app/components/mode-toggle";
import { Link } from "react-router";

export default function AppTopBar() {
  return (
    <div className='w-full border-b border-gray-250 h-12 flex flex-row items-center px-4 gap-2'>
      <SidebarTrigger />

      <Link className='text-lg flex items-center ml-2' to='/'>
        L<img src='/o.png' alt='O' className='inline h-6' />NG&nbsp;<span className='hidden sm:block'>Hub</span>
      </Link>

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
