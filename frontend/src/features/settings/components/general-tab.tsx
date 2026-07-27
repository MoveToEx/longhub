import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/shared/components/ui/field";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/shared/components/ui/select";
import usePreference from "@/shared/hooks/use-preference";
import { CodeIcon, ImageIcon, Split } from "lucide-react";
import { useMemo } from "react";

function CopyMode() {
  const [preference, setPreference] = usePreference();

  const items = useMemo(() => [
    {
      label: <span className='flex flex-row items-center gap-2'><Split className='inline' /> Auto</span>,
      value: 'auto'
    },
    {
      label: <span className='flex flex-row items-center gap-2'><ImageIcon className='inline' /> PNG</span>,
      value: 'png'
    },
    {
      label: <span className='flex flex-row items-center gap-2'><CodeIcon className='inline' /> HTML</span>,
      value: 'html'
    },
  ], []);

  return (
    <Field className='*:w-fit'>
      <FieldLabel>
        Copy mode
      </FieldLabel>
      <Select
        items={items}
        value={preference.copyMode}
        onValueChange={value => {
          setPreference(current => ({
            ...current,
            copyMode: value ?? 'png',
          }));
        }}>
        <SelectTrigger className='min-w-36'>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {items.map((item) => (
            <SelectItem key={item.value} value={item.value}>
              {item.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <FieldDescription>
        PNG is recommended in most cases, but HTML <b>may</b> allow you to copy GIF into other apps.
        <br />
        Auto mode uses HTML only for GIF and PNG for other formats.
      </FieldDescription>
    </Field>
  )
}

export default function GeneralTab() {

  return (
    <div className='w-full flex flex-col gap-2 items-center'>
      <div className='w-full md:w-3/4'>
        <span className='text-lg'>Copying</span>

        <FieldGroup className='mt-4'>
          <CopyMode />
        </FieldGroup>
      </div>
    </div>
  )
}
