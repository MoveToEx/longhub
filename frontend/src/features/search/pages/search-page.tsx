import { useMemo, useRef, useState, type FormEvent } from "react";
import { ListFilter, Plus, Search, Trash2 } from "lucide-react";
import { useSearchParams } from "react-router";
import { Alert, AlertDescription, AlertTitle } from "@/shared/components/ui/alert";
import { Button } from "@/shared/components/ui/button";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/shared/components/ui/empty";
import { Input } from "@/shared/components/ui/input";
import { Label } from "@/shared/components/ui/label";
import { Pagination } from "@/shared/components/ui/pagination";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/shared/components/ui/select";
import { Spinner } from "@/shared/components/ui/spinner";
import ImageGrid from "@/features/images/components/image-grid";
import TagInput from "@/features/tags/components/tag-input";
import useSearch, {
  type SearchCondition,
  type SearchConditionType,
  type SearchOrder,
  type SearchOrderBy,
  type SearchRequest,
} from "@/features/search/hooks/use-search";
import type { Rating } from "@/shared/lib/types";

const PAGE_SIZE = 24;

type DraftCondition = SearchCondition & { id: number };

const conditionOptions: { value: SearchConditionType; label: string }[] = [
  { value: 'tagInclude', label: 'Tag includes' },
  { value: 'tagExclude', label: 'Tag excludes' },
  { value: 'ratingEq', label: 'Rating equals' },
  { value: 'textContains', label: 'Text contains' },
  { value: 'uploadedBy', label: 'Uploaded by' },
];

const orderByOptions: { value: SearchOrderBy; label: string }[] = [
  { value: 'uploadDate', label: 'Upload date' },
  { value: 'id', label: 'Image ID' },
];

const orderOptions: { value: SearchOrder; label: string }[] = [
  { value: 'desc', label: 'Descending' },
  { value: 'asc', label: 'Ascending' },
];

const ratingOptions: { value: Rating; label: string }[] = [
  { value: 'none', label: 'None' },
  { value: 'moderate', label: 'Moderate' },
  { value: 'violent', label: 'Violent' },
];

const conditionPlaceholder: Record<SearchConditionType, string> = {
  tagInclude: 'landscape',
  tagExclude: 'monochrome',
  ratingEq: '',
  textContains: 'Words or phrase',
  uploadedBy: 'Username or user ID',
};

export default function SearchPage() {
  const nextConditionID = useRef(1);
  const [searchParams, setSearchParams] = useSearchParams({ p: '1' });
  const page = useMemo(
    () => Math.max(1, Number(searchParams.get('p') ?? 1) || 1),
    [searchParams],
  );
  const [conditions, setConditions] = useState<DraftCondition[]>([]);
  const [orderBy, setOrderBy] = useState<SearchOrderBy>('uploadDate');
  const [order, setOrder] = useState<SearchOrder>('desc');
  const [request, setRequest] = useState<SearchRequest>({
    conditions: [],
    orderBy: 'uploadDate',
    order: 'desc',
  });

  const { data, error, isLoading } = useSearch(request, (page - 1) * PAGE_SIZE, PAGE_SIZE);

  const updateCondition = (id: number, patch: Partial<SearchCondition>) => {
    setConditions(current => current.map(condition => {
      if (condition.id !== id) return condition;
      const next = { ...condition, ...patch };
      if (patch.type === 'ratingEq' && condition.type !== 'ratingEq') next.value = 'none';
      if (patch.type && patch.type !== 'ratingEq' && condition.type === 'ratingEq') next.value = '';
      return next;
    }));
  };

  const submit = (event: FormEvent) => {
    event.preventDefault();
    setSearchParams({ p: '1' });
    setRequest({
      conditions: conditions.map(({ type, value }) => ({ type, value: value.trim() })),
      orderBy,
      order,
    });
  };

  return (
    <div className='w-full flex flex-col gap-6'>
      <div className='text-lg'>Conditions</div>
      <form className='flex flex-col' onSubmit={submit}>
        {conditions.length === 0 && (
          <div className='rounded-lg border border-dashed p-5 text-center text-sm text-muted-foreground'>
            No conditions yet. Searching now will show all indexed images.
          </div>
        )}

        <div className='pb-4'>
          {conditions.map((condition, index) => (
            <div key={condition.id} className='flex flex-col gap-2 p-2 md:flex-row md:items-center'>
              <span className='w-12 shrink-0 text-xs font-medium uppercase tracking-wide text-muted-foreground'>
                {index === 0 ? 'Where' : 'And'}
              </span>
              <Select
                items={conditionOptions}
                value={condition.type}
                onValueChange={value => value && updateCondition(condition.id, { type: value as SearchConditionType })}>
                <SelectTrigger className='w-full md:w-44'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {conditionOptions.map(option => (
                    <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>

              {condition.type === 'ratingEq' ? (
                <Select
                  items={ratingOptions}
                  value={condition.value}
                  onValueChange={value => value && updateCondition(condition.id, { value })}>
                  <SelectTrigger className='w-full md:flex-1'>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {ratingOptions.map(option => (
                      <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : condition.type === 'tagInclude' || condition.type === 'tagExclude' ? (
                <TagInput
                  required
                  className='md:flex-1'
                  placeholder={conditionPlaceholder[condition.type]}
                  value={condition.value}
                  onValueChange={value => updateCondition(condition.id, { value })}
                  onTagSelect={value => updateCondition(condition.id, { value })}
                />
              ) : (
                <Input
                  required
                  className='md:flex-1'
                  placeholder={conditionPlaceholder[condition.type]}
                  value={condition.value}
                  onChange={event => updateCondition(condition.id, { value: event.target.value })}
                />
              )}

              <Button
                type='button'
                variant='ghost'
                size='icon'
                aria-label='Remove condition'
                onClick={() => setConditions(current => current.filter(item => item.id !== condition.id))}>
                <Trash2 />
              </Button>
            </div>
          ))}
        </div>

        <div className='flex flex-wrap items-end gap-3'>
          <Button
            type='button'
            variant='outline'
            onClick={() => setConditions(current => [
              ...current,
              { id: nextConditionID.current++, type: 'tagInclude', value: '' },
            ])}>
            <Plus /> Add condition
          </Button>

          <div className='ml-0 flex flex-wrap gap-3 md:ml-auto'>
            <div className='flex flex-col gap-1.5'>
              <Label>Order by</Label>
              <Select
                items={orderByOptions}
                value={orderBy}
                onValueChange={value => value && setOrderBy(value as SearchOrderBy)}>
                <SelectTrigger className='w-36'><SelectValue /></SelectTrigger>
                <SelectContent>
                  {orderByOptions.map(option => (
                    <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className='flex flex-col gap-1.5'>
              <Label>Direction</Label>
              <Select
                items={orderOptions}
                value={order}
                onValueChange={value => value && setOrder(value as SearchOrder)}>
                <SelectTrigger className='w-36'><SelectValue /></SelectTrigger>
                <SelectContent>
                  {orderOptions.map(option => (
                    <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <Button className='self-end' type='submit'>
              <Search /> Search
            </Button>
          </div>
        </div>
      </form>

      {isLoading && (
        <div className='flex h-32 items-center justify-center gap-2'><Spinner /> Loading</div>
      )}

      {error && (
        <Alert variant='destructive'>
          <AlertTitle>Search failed</AlertTitle>
          <AlertDescription>The image index could not process these conditions. Please try again.</AlertDescription>
        </Alert>
      )}

      {!isLoading && !error && data?.total === 0 && (
        <Empty className='h-64'>
          <EmptyHeader>
            <EmptyMedia variant='icon'><ListFilter /></EmptyMedia>
            <EmptyTitle>No matching images</EmptyTitle>
            <EmptyDescription>Try removing or changing one of the conditions.</EmptyDescription>
          </EmptyHeader>
        </Empty>
      )}

      {!isLoading && !error && data && data.total > 0 && (
        <>
          <div className='text-sm text-muted-foreground'>{data.total} matching images</div>
          <ImageGrid items={data.images} />
          <Pagination
            className='mt-2 mb-2'
            count={Math.ceil(data.total / PAGE_SIZE)}
            page={page}
            onChange={(_, nextPage) => setSearchParams({ p: nextPage.toString() })}
          />
        </>
      )}
    </div>
  );
}
