import useUser from "@/hooks/server/use-user"
import { Spinner } from "@/components/ui/spinner";
import { ArrowUpToLine, Calendar } from "lucide-react";


export default function UserInfo({ id }: {
  id: number
}) {
  const { data, isLoading } = useUser(id);

  return (
    <div>
      {isLoading && (
        <div className='w-full h-32 flex items-center justify-center gap-2'>
          <Spinner /> Loading
        </div>
      )}

      {data && (
        <ul className='text-accent-foreground text-sm *:flex *:flex-row *:justify-start *:items-center *:gap-2'>
          <li><Calendar className='inline' size={12} /> Joined on {new Date(data.createdAt).toLocaleDateString()}</li>
          <li><ArrowUpToLine className='inline' size={12} /> {data.images + data.versions} contributions so far</li>
        </ul>
      )}
    </div>
  )
}