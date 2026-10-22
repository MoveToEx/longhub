import { Button } from "@/shared/components/ui/button";
import { Spinner } from "@/shared/components/ui/spinner";
import { copyImage, formatError } from "@/shared/lib/utils";
import { Copy } from "lucide-react";
import { useState, type ReactNode } from "react";
import { toast } from "sonner";

export default function ImageGridItem({ imageUrl, children }: {
  imageUrl: string,
  children: ReactNode,
}) {
  const [copying, setCopying] = useState(false);

  return (
    <div className='group/image relative min-w-0'>
      {children}
      <Button
        variant='outline'
        size='icon-sm'
        className='absolute top-2 right-2 bg-background opacity-0 pointer-events-none group-hover/image:opacity-100 group-hover/image:pointer-events-auto group-focus-within/image:opacity-100 group-focus-within/image:pointer-events-auto aria-busy:opacity-100 dark:bg-background [@media(hover:none)]:opacity-100 [@media(hover:none)]:pointer-events-auto'
        aria-label='Copy image'
        title='Copy image'
        aria-busy={copying}
        disabled={copying}
        onClick={async () => {
          setCopying(true);
          try {
            await copyImage(imageUrl);
            toast.success('Copied to clipboard');
          }
          catch (error) {
            toast.error(formatError(error));
          }
          finally {
            setCopying(false);
          }
        }}>
        {copying ? <Spinner /> : <Copy />}
      </Button>
    </div>
  );
}
