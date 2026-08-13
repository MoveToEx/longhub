import { Spinner } from "@/shared/components/ui/spinner";
import useAuth from "@/features/auth/hooks/use-auth";
import useFavoriteTags from "@/features/favorites/hooks/use-favorite-tags";
import useImages from "@/features/images/hooks/use-images";
import useRandomImages from "@/features/images/hooks/use-random-images";
import type { Image, Tag } from "@/shared/lib/types";
import { Link } from "react-router";

function ImageGrid({ items }: {
  items: Pick<Image, 'id' | 'imageUrl'>[]
}) {
  return (
    <div className='grid w-full grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-6'>
      {items.map(({ id, imageUrl }) => (
        <Link key={id} to={`/image/${id}`}>
          <img className='object-contain object-center w-full h-32' src={imageUrl} alt='Image' crossOrigin="anonymous" />
        </Link>
      ))}
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
    <ImageGrid items={data.slice(0, 6)} />
  )
}

function RecommendedImages({ data, isLoading }: {
  data: (Tag & { count: number })[] | undefined,
  isLoading: boolean,
}) {
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
            #{it.name}
          </div>
          <RecommendedImagesRow tag={it.name} />
        </div>
      ))}
    </div>
  )
}

function Recent({ expanded }: { expanded: boolean }) {
  const { data, isLoading } = useImages(0, 24);

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

  return <ImageGrid items={data.images.slice(0, expanded ? 24 : 6)} />
}

function PageSections({ recommendations, expandRecent }: {
  recommendations: React.ReactNode,
  expandRecent: boolean,
}) {
  return (
    <div className='w-full flex flex-col items-center'>
      <div className='w-full flex flex-col items-start gap-8'>
        <div className='w-full flex flex-col items-start gap-2'>
          <div className='text-2xl'>
            Based on your favorites
          </div>
          <div className='ml-2 w-full'>
            {recommendations}
          </div>
        </div>
        <div className='w-full flex flex-col items-start gap-2'>
          <div className='text-2xl'>
            Recent uploads
          </div>
          <Recent expanded={expandRecent} />
        </div>
      </div>
    </div>
  )
}

function AuthenticatedPage() {
  const { data, isLoading } = useFavoriteTags();

  return (
    <PageSections
      recommendations={<RecommendedImages data={data} isLoading={isLoading} />}
      expandRecent={!isLoading && data?.length === 0}
    />
  )
}

export default function IndexPage() {
  const user = useAuth();

  if (user.isLoading) {
    return (
      <PageSections
        recommendations={(
          <div className='w-full h-32 flex flex-row justify-center items-center gap-2'>
            <Spinner /> Loading...
          </div>
        )}
        expandRecent={false}
      />
    )
  }

  if (!user.data) {
    return (
      <PageSections
        recommendations={(
          <div className='w-full h-32 flex flex-row justify-center items-center gap-2'>
            Login to show recommendations
          </div>
        )}
        expandRecent
      />
    )
  }

  return <AuthenticatedPage />
}
