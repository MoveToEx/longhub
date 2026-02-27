import { Carousel, CarouselContent, CarouselItem } from "@/components/ui/carousel";
import { Spinner } from "@/components/ui/spinner";
import useAuth from "@/hooks/server/use-auth";
import useFavoriteTags from "@/hooks/server/use-favorite-tags";
import useImages from "@/hooks/server/use-images";
import useRandomImages from "@/hooks/server/use-random-images";
import type { Image } from "@/lib/types";

function ImageCarousal({ items }: {
  items: Pick<Image, 'id' | 'imageUrl'>[]
}) {
  return (
    <div className='w-full h-full'>
      <Carousel className='w-full h-full'>
        <CarouselContent className='w-full h-full'>
          {items.map(({ id, imageUrl }) => (
            <CarouselItem
              key={id}
              className='lg:basis-1/6 md:basis-1/3 basis-1/2'>
              <a href={`/#/image/${id}`}>
                <img className='object-contain object-center' src={imageUrl} alt='Image' crossOrigin="anonymous" />
              </a>
            </CarouselItem>
          ))}
        </CarouselContent>
      </Carousel>
    </div>
  )
}

function RecommendedImagesRow({ tag }: { tag: string }) {
  const { data, isLoading } = useRandomImages(tag);

  if (isLoading) {
    return (
      <div className='w-full h-32 flex flex-row justify-center items-center gap-2'>
        <Spinner /> Loading
      </div>
    )
  }

  if (!data) {
    return (
      <div className='w-full flex flex-row items-center gap-2'>
        Failed
      </div>
    )
  }

  return (
    <ImageCarousal items={data} />
  )
}

function RecommendedImages() {
  const { data, isLoading } = useFavoriteTags();

  if (isLoading) {
    return (
      <div className='w-full h-32 flex flex-row justify-center items-center gap-2'>
        <Spinner /> Loading
      </div>
    )
  }

  if (data === undefined) return <></>

  if (data.length === 0) {
    return (
      <div className='w-full h-32 flex flex-row justify-center items-center gap-2'>
        No favorites yet
      </div>
    )
  }

  return (
    <div className='flex flex-col items-start gap-4'>
      {data.map(it => (
        <div key={it.id} className='w-full flex flex-col'>
          <div className='text-xl'>
            Images under #{it.name}
          </div>
          <RecommendedImagesRow tag={it.name} />
        </div>
      ))}
    </div>
  )
}

function Recommend() {
  const user = useAuth();

  if (!user.isLoading && !user.data) {
    return (
      <div className='w-full h-32 flex flex-row justify-center items-center gap-2'>
        Login to show recommendations
      </div>
    )
  }

  if (user.isLoading) {
    return (
      <div className='w-full h-32 flex flex-row justify-center items-center gap-2'>
        <Spinner /> Loading...
      </div>
    )
  }

  return (
    <RecommendedImages />
  )
}

function Recent() {
  const { data, isLoading } = useImages(0, 12);

  if (isLoading) {
    return (
      <div className='flex flex-row items-center gap-2'>
        <Spinner /> Loading...
      </div>
    )
  }

  if (!data) {
    return (
      <div className='flex flex-row items-center gap-2'>
        Failed
      </div>
    )
  }

  return <ImageCarousal items={data.images} />
}

export default function IndexPage() {
  return (
    <div className='w-full flex flex-col items-center'>
      <div className='w-full flex flex-col items-start gap-8'>
        <div className='w-full flex flex-col items-start gap-2'>
          <div className='text-2xl'>
            Based on your favorites
          </div>
          <div className='ml-2 w-full'>
            <Recommend />
          </div>
        </div>
        <div className='w-full flex flex-col items-start gap-2'>
          <div className='text-2xl'>
            Recent uploads
          </div>
          <Recent />
        </div>
      </div>
    </div>
  )
}