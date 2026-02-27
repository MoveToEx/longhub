import useImages from "@/hooks/server/use-images";
import { Carousel, CarouselContent, CarouselItem } from "./ui/carousel";
import { Spinner } from "./ui/spinner";


export default function ImageCarousal() {
  const { data, isLoading } = useImages(0, 12);
  return (
    <div className='w-full flex flex-col items-center justify-center'>
      {isLoading && (
        <div className='flex flex-row items-center gap-2'>
          <Spinner /> Loading...
        </div>
      )}

      {data && (
        <Carousel>
          <CarouselContent>
            {data.images.map(it => (
              <CarouselItem
                key={it.id}
                className='lg:basis-1/6 md:basis-1/3 basis-1/2'>
                <img src={it.imageUrl} alt='Image' crossOrigin="anonymous"/>
              </CarouselItem>
            ))}
          </CarouselContent>
        </Carousel>
      )}
    </div>
  )
}