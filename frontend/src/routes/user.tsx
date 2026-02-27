import ContributionGrid from "@/components/user/contribution-grid";
import { Separator } from "@/components/ui/separator";
import UserHeader from "@/components/user/header";
import UserInfo from "@/components/user/info";
import { useParams } from "react-router"
import UserImages from "@/components/user/images";


export default function UserPage() {
  const { id } = useParams();

  if (id === undefined || !Number.isInteger(Number(id))) {
    throw new Error();
  }

  return (
    <div className='w-full flex flex-col gap-4'>
      <div className='h-6'>
        <UserHeader id={Number(id)} />
      </div>
      <div className='flex flex-col md:flex-row gap-4'>
        <div className='md:flex-2'>
          <ContributionGrid userId={Number(id)} />
        </div>
        <Separator className='hidden md:block' orientation='vertical' />
        <div className='md:flex-1 py-2'>
          <UserInfo id={Number(id)} />
        </div>
      </div>
      <div>
        <UserImages id={Number(id)} />
      </div>
    </div>
  )
}