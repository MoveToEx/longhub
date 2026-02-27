import useUser from "@/hooks/server/use-user"
import { Spinner } from "@/components/ui/spinner";
import { User } from "lucide-react";

export default function UserHeader({ id }: {
  id: number
}) {
  const { data, isLoading } = useUser(id);
  return (
    <div className='w-full h-full'>
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