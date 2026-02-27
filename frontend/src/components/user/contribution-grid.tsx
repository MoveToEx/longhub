import useUserContribution from "@/hooks/server/use-user-contribution"
import { Spinner } from "@/components/ui/spinner";
import { useMemo } from "react";
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";

const COLOR_SCALE = [
  "bg-gray-100 dark:bg-gray-800",
  "bg-green-200",
  "bg-green-400",
  "bg-green-600",
  "bg-green-800",
];


function getColor(percentage: number) {
  if (percentage < 0.0001) return COLOR_SCALE[0];
  if (percentage <= 0.25) return COLOR_SCALE[1];
  if (percentage <= 0.5) return COLOR_SCALE[2];
  if (percentage <= 0.75) return COLOR_SCALE[3];
  return COLOR_SCALE[4];
}

export default function ContributionGrid({
  userId
}: {
  userId: number
}) {
  const { data, isLoading } = useUserContribution(userId);
  const start = useMemo(() => {
    const d = new Date();
    d.setFullYear(d.getFullYear() - 1);
    d.setDate(d.getDate() + 1);
    return d;
  }, []);

  const maximum = useMemo(() => {
    if (!data) return 1;

    return data.map(it => it.imageCount + it.versionCount).reduce((prev, cur) => prev > cur ? prev : cur);
  }, [data]);

  const days = useMemo(() => {
    const today = new Date();
    const start = new Date(today);
    start.setFullYear(today.getFullYear() - 1);
    start.setDate(start.getDate() + 1);


    const result: (Date | null)[] = [];


    const offset = start.getDay();
    for (let i = 0; i < offset; i++) {
      result.push(null);
    }


    for (let i = 0; i < 365; i++) {
      const d = new Date(start);
      d.setDate(start.getDate() + i);
      result.push(d);
    }


    return result;
  }, []);

  const map = useMemo(() => {
    if (!data) return new Map() as Map<string, Pick<NonNullable<typeof data>[number], 'imageCount' | 'versionCount'>>;

    const m: Map<string, Pick<NonNullable<typeof data>[number], 'imageCount' | 'versionCount'>> = new Map();
    for (const it of data) {
      const { day, ...rest } = it;
      m.set(day, rest);
    }
    return m;
  }, [data]);


  return (
    <ScrollArea className='rounded-md max-w-full h-32 border border-gray-250 overflow-x-auto px-4'>
      {isLoading || !data && (
        <div className='w-full h-full flex flex-row justify-center items-center'>
          <Spinner /> Loading
        </div>
      )}
      {data && (
        <div className='w-full h-full flex flex-row gap-0.5 justify-center items-center'>
          <div className='h-full flex flex-col justify-center items-end text-[10px] gap-0.5 *:h-2.5 mr-2'>
            <span></span>
            <span>Mon</span>
            <span></span>
            <span>Wed</span>
            <span></span>
            <span>Fri</span>
            <span></span>
          </div>
          {Array.from({ length: 53 }).map((_, week) => (
            <div key={week} className='flex flex-col gap-0.5'>
              {week === 0 && Array.from({ length: start.getDay() })}
              {Array.from({ length: 7 }).map((_, day) => {
                const index = week * 7 + day;
                const date = days[index];
                if (!date) return <div key={day} className="w-2.5 h-2.5 rounded-xs bg-transparent" />;

                const iso = date.toISOString().slice(0, 10);
                const it = map.get(iso);

                let count;

                if (!it) count = 0;
                else count = it.imageCount + it.versionCount;

                return (
                  <div
                    key={day}
                    title={`${it?.imageCount ?? 0} images, ${it?.versionCount ?? 0} edits on ${iso}`}
                    className={`w-2.5 h-2.5 rounded-xs border border-gray-150 ${getColor(count / maximum)}`}
                  />
                );
              })}
            </div>
          ))}
        </div>
      )}
      <ScrollBar orientation='horizontal' />
    </ScrollArea>
  )
}