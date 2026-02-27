import { Pagination, PaginationContent, PaginationItem, PaginationLink, PaginationNext, PaginationPrevious } from "@/components/ui/pagination";
import { Spinner } from "@/components/ui/spinner";
import useImages from "@/hooks/server/use-images";
import { useMemo } from "react";
import { useSearchParams } from "react-router";
import ImageGrid from "@/components/image-grid";

export default function ImagesPage() {
  const [searchParams, setSearchParams] = useSearchParams({
    p: "1"
  });
  const page = useMemo(() => Number(searchParams.get('p') ?? 1), [searchParams]);

  const { data, isLoading } = useImages((page - 1) * 48);

  return (
    <div className='w-full h-full'>
      {isLoading && (
        <div className='w-full h-32 flex flex-col justify-center items-center gap-2'>
          <Spinner />
          Loading
        </div>
      )}
      {data && <ImageGrid items={data.images} />}

      <Pagination className='mt-4 mb-2'>
        <PaginationContent>
          <PaginationItem>
            <PaginationPrevious
              onClick={() => {
                if (page === 1) return;

                setSearchParams({
                  p: (page - 1).toString()
                });
              }} />
          </PaginationItem>

          {page > 2 && (
            <PaginationItem>
              <PaginationLink onClick={() => {
                setSearchParams({
                  p: (page - 1).toString()
                })
              }}>{page - 1}</PaginationLink>
            </PaginationItem>
          )}

          <PaginationItem>
            <PaginationLink isActive>{page}</PaginationLink>
          </PaginationItem>

          <PaginationItem>
            <PaginationNext
              onClick={() => {
                if (page * 48 >= (data?.total ?? 0)) {
                  return;
                }

                setSearchParams({
                  p: String(page + 1)
                })
              }} />
          </PaginationItem>
        </PaginationContent>
      </Pagination>
    </div>
  )
}