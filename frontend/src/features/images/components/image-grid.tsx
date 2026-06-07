import { Button } from "@/shared/components/ui/button";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/shared/components/ui/hover-card";
import { Spinner } from "@/shared/components/ui/spinner";
import { Copy } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { copyImage } from "@/shared/lib/utils";
import type { Image } from "@/shared/lib/types";
import LikeButton from "./like-button";
import { Link } from "react-router";

type GridItem = Pick<Image, 'id' | 'imageUrl'>;

function CopyButton({
  url
}: {
  url: string,
}) {
  const [loading, setLoading] = useState(false);

  return (
    <Button
      variant='outline'
      disabled={loading}
      onClick={async () => {
        setLoading(true);
        try {
          await copyImage(url);
          toast.success('Copied to clipboard');
        }
        catch (e) {
          if (e instanceof Error) {
            toast.error(e.message);
          }
        }
        finally {
          setLoading(false);
        }
      }}>
      {loading && <Spinner />}
      {loading || <Copy />}
      Copy
    </Button>
  )
}

function Image({ id, imageUrl }: {
  id: number,
  imageUrl: string,
}) {
  return (
    <HoverCard>
      <HoverCardTrigger delay={100} closeDelay={100} render={
        <Link to={`/image/${id}`}>
          <img src={imageUrl} className='w-full h-48 object-contain object-center' crossOrigin="anonymous" />
        </Link>
      } />
      <HoverCardContent className='flex flex-row justify-evenly'>
        <LikeButton id={id} />
        <CopyButton url={imageUrl} />
      </HoverCardContent>
    </HoverCard>
  )
}

export default function ImageGrid({ items }: {
  items: GridItem[]
}) {
  return (
    <div className='w-full h-full'>
      <div className='grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-2'>
        {items.map(it => (
          <div key={it.id}>
            <Image id={it.id} imageUrl={it.imageUrl} />
          </div>
        ))}
      </div>
    </div>
  )
}
