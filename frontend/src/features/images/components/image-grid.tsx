import type { Image } from "@/shared/lib/types";
import { Link } from "react-router";
import ImageGridItem from "./image-grid-item";

type GridItem = Pick<Image, 'id' | 'imageUrl'>;

export default function ImageGrid({ items }: {
  items: GridItem[]
}) {
  return (
    <div className='w-full h-full'>
      <div className='grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-2'>
        {items.map(({ id, imageUrl }) => (
          <ImageGridItem key={id} imageUrl={imageUrl}>
            <Link to={`/image/${id}`}>
              <img src={imageUrl} alt='Image' className='w-full h-48 object-contain object-center' crossOrigin="anonymous" />
            </Link>
          </ImageGridItem>
        ))}
      </div>
    </div>
  )
}
