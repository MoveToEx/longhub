import { Pagination } from "@/shared/components/ui/pagination";
import { Spinner } from "@/shared/components/ui/spinner";
import useImages from "@/features/images/hooks/use-images";
import { useMemo } from "react";
import { useSearchParams } from "react-router";
import ImageGrid from "@/features/images/components/image-grid";

const PAGE_SIZE = 24;

export default function ImagesPage() {
  const [searchParams, setSearchParams] = useSearchParams({
    p: "1"
  });
  const page = useMemo(() => Math.max(1, Number(searchParams.get('p') ?? 1) || 1), [searchParams]);

  const { data, isLoading } = useImages((page - 1) * PAGE_SIZE, PAGE_SIZE);

  return (
    <div className='w-full h-full'>
      {isLoading && (
        <div className='w-full h-32 flex flex-col justify-center items-center gap-2'>
          <Spinner />
          Loading
        </div>
      )}
      {data && <ImageGrid items={data.images} />}

      <Pagination
        className='mt-4 mb-2'
        count={Math.ceil((data?.total ?? 0) / PAGE_SIZE)}
        page={page}
        onChange={(_, page) => setSearchParams({ p: page.toString() })}
      />
    </div>
  )
}
