import { Button } from "@/components/ui/button";
import { Dialog as BaseDialog } from "@base-ui/react";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { Spinner } from "@/components/ui/spinner";
import { Copy } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import EditShortcutDialog, { type Payload as ShortcutPayload } from "@/components/dialogs/edit-shortcut";
import { copyImage } from "@/lib/utils";
import type { Image } from "@/lib/types";
import LikeButton from "./like-button";

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

function Image({ id, imageUrl, onCustomize }: {
  id: number,
  imageUrl: string,
  onCustomize: () => void
}) {
  return (
    <HoverCard>
      <HoverCardTrigger delay={500} closeDelay={200} render={
        <a href={`/#/image/${id}`}>
          <img src={imageUrl} className='w-full h-48 object-contain object-center' crossOrigin="anonymous" />
        </a>
      } />
      <HoverCardContent className='flex flex-row justify-evenly'>
        <LikeButton
          id={id}
          customizeAction={() => onCustomize()}/>
        <CopyButton url={imageUrl} />
      </HoverCardContent>
    </HoverCard>
  )
}

const customizeHandle = BaseDialog.createHandle<ShortcutPayload>();

export default function ImageGrid({ items }: {
  items: GridItem[]
}) {
  return (
    <div className='w-full h-full'>
      <EditShortcutDialog handle={customizeHandle} />
      <div className='grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-2'>
        {items.map(it => (
          <div key={it.id}>
            <Image id={it.id} imageUrl={it.imageUrl} onCustomize={() => {
              customizeHandle.openWithPayload({ ...it, shortcut: '' });
            }} />
          </div>
        ))}
      </div>
    </div>
  )
}