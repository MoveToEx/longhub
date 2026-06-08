import useUser from "@/features/user/hooks/use-user"
import { Spinner } from "@/shared/components/ui/spinner";
import { User } from "lucide-react";

export default function UserHeader({ id }: {
  id: number
}) {
  const { data, isLoading } = useUser(id);
  return (
    <div className='w-full'>
      {isLoading && (
        <Spinner />
      )}

      {data && (
        <div className='flex flex-row items-center justify-start gap-4'>
          <User />
          @{data.username}
        </div>
      )}
    </div>
  )
}