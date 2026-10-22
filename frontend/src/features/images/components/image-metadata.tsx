import Tags from "@/features/tags/components/tags";
import type { Rating } from "@/shared/lib/types";
import { Hash, Pilcrow, ShieldAlert } from "lucide-react";

export default function ImageMetadata({ text, tags, rating }: {
  text: string,
  tags: string[],
  rating: Rating,
}) {
  return (
    <table className='w-full table-fixed border-collapse [overflow-wrap:anywhere]'>
      <colgroup>
        <col className='w-5' />
        <col />
      </colgroup>
      <tbody>
        <tr>
          <td className='pt-2 align-top'>
            <Pilcrow size={12} />
          </td>
          <td className='py-1 align-top'>
            {text || <span className='italic'>(no text)</span>}
          </td>
        </tr>
        <tr>
          <td className='py-1 align-top'>
            <div className={`flex h-lh items-center ${tags.length > 0 ? 'box-content border-y border-transparent py-1' : ''}`}>
              <Hash size={12} />
            </div>
          </td>
          <td className='py-1 align-top'>
            <div className='flex flex-wrap items-center gap-1'>
              <Tags value={tags} />
              {tags.length === 0 && (
                <span className='italic text-accent-foreground'>(No tags)</span>
              )}
            </div>
          </td>
        </tr>
        <tr>
          <td className='pt-2 align-top'>
            <ShieldAlert size={12} />
          </td>
          <td className='py-1 align-top'>
            Rated as {rating}
          </td>
        </tr>
      </tbody>
    </table>
  );
}
