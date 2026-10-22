import { Button } from "@/shared/components/ui/button";
import { Card, CardContent, CardFooter } from "@/shared/components/ui/card";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/shared/components/ui/collapsible";
import { Spinner } from "@/shared/components/ui/spinner";
import useImage from "@/features/images/hooks/use-image";
import useImageVersions from "@/features/images/hooks/use-image-versions";
import { Check, ChevronDownIcon, Copy, Download, Edit, Hash, Pilcrow, ShieldAlert } from "lucide-react";
import { useParams } from "react-router"
import pluralize from 'pluralize'
import { ScrollArea } from "@/shared/components/ui/scroll-area";
import { useEffect, useMemo, useState } from "react";
import api from "@/shared/lib/axios";
import { Dialog, DialogClose, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/shared/components/ui/dialog";
import { Dialog as BaseDialog } from '@base-ui/react'
import { toast } from "sonner";
import { mutate } from "@/shared/lib/swr";
import Tags from "@/features/tags/components/tags";
import z from "zod";
import { Controller, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Field, FieldError, FieldLabel } from "@/shared/components/ui/field";
import { Input } from "@/shared/components/ui/input";
import TagsInput from "@/features/tags/components/tags-input";
import { ToggleGroup, ToggleGroupItem } from "@/shared/components/ui/toggle-group";
import type { Rating } from "@/shared/lib/types";
import _ from "lodash";
import LikeButton from "../components/like-button";
import { copyImage, formatError } from "@/shared/lib/utils";


type Payload = {
  id: number,
  rating: string,
  tags: string[],
  text: string,
}

const editSchema = z.object({
  text: z.string().refine(val => val.indexOf(' ') === -1, 'Text should not contain spaces'),
  tags: z.string().array(),
  rating: z.enum(['none', 'moderate', 'violent'])
});

function EditDialog({ handle }: { handle: BaseDialog.Handle<Payload> }) {
  const [loading, setLoading] = useState(false);

  const submit = async ({ id, text, rating, tags }: Payload) => {
    if (!Number.isSafeInteger(id)) return;

    setLoading(true);

    try {
      await api.patch(`/image/${id}`, { text, rating, tags });
      await mutate('image');
      handle.close();
      toast.success('Success');
    }
    catch (e) {
      toast.error(formatError(e));
    }
    finally {
      setLoading(false);
    }
  }

  return (
    <Dialog<Payload> handle={handle}>
      {function Content({ payload }) {
        const form = useForm<z.infer<typeof editSchema>>({
          resolver: zodResolver(editSchema),
          defaultValues: useMemo(() => {
            if (!payload) {
              return {
                text: '',
                tags: [],
                rating: 'none'
              }
            }
            return {
              text: payload.text,
              tags: payload.tags,
              rating: payload.rating as Rating
            }
          }, [payload])
        });

        useEffect(() => {
          if (!payload) return;

          if (!_.isEqual(payload, form.getValues())) {
            form.reset({
              text: payload.text,
              tags: payload.tags,
              rating: payload.rating as Rating
            });
          }
        }, [payload, form]);

        return (
          <DialogContent>
            <DialogHeader>
              <DialogTitle className='text-xl'>
                Edit image metadata
              </DialogTitle>
            </DialogHeader>

            <form className='flex flex-col gap-4' onSubmit={form.handleSubmit(data => submit({ ...data, id: payload?.id ?? 0 }))}>
              <Controller
                name='text'
                control={form.control}
                render={({ field, fieldState }) => (
                  <Field data-invalid={fieldState.invalid}>
                    <FieldLabel htmlFor="form-upload-text">
                      Text
                    </FieldLabel>
                    <Input
                      {...field}
                      id="form-upload-text"
                    />
                    {fieldState.invalid && (
                      <FieldError errors={[fieldState.error]} />
                    )}
                  </Field>
                )}>
              </Controller>
              <Controller
                name='tags'
                control={form.control}
                render={({ field, fieldState }) => (
                  <Field data-invalid={fieldState.invalid}>
                    <FieldLabel>
                      Tags
                    </FieldLabel>

                    <TagsInput {...field} />

                    {fieldState.invalid && (
                      <FieldError errors={[fieldState.error]} />
                    )}
                  </Field>
                )} />
              <Controller
                name='rating'
                control={form.control}
                render={({ field, fieldState }) => (
                  <Field data-invalid={fieldState.invalid} className='*:w-fit'>
                    <FieldLabel>
                      Rating
                    </FieldLabel>
                    <ToggleGroup
                      {...field}
                      className='rounded-md shadow-xs border overflow-hidden *:not-last:border-r'
                      value={[field.value]}
                      onValueChange={val => form.setValue('rating', val[0] as Rating)}>
                      <ToggleGroupItem value="none">None</ToggleGroupItem>
                      <ToggleGroupItem value="moderate">Moderate</ToggleGroupItem>
                      <ToggleGroupItem value="violent">Violent</ToggleGroupItem>
                    </ToggleGroup>
                    {fieldState.invalid && (
                      <FieldError errors={[fieldState.error]} />
                    )}
                  </Field>
                )}>
              </Controller>

              <DialogFooter>
                <Button
                  disabled={loading}
                  type='submit'>
                  {loading && <Spinner />}
                  {loading || <Check />} Submit
                </Button>
                <DialogClose
                  disabled={loading}
                  render={
                    <Button variant='outline'>
                      Cancel
                    </Button>
                  } />
              </DialogFooter>
            </form>
          </DialogContent>
        )
      }}
    </Dialog>
  )
}

function Image({ id }: { id: number }) {
  const { data, isLoading } = useImage(id);

  if (isLoading) {
    return (
      <div className='w-full h-32 flex flex-col items-center justify-center gap-2'>
        <Spinner /> Loading...
      </div>
    )
  }

  return (
    <div className='w-full h-auto flex flex-col justify-center items-center gap-2'>
      {data && (
        <img
          src={data.imageUrl}
          className='w-full h-full object-contain object-center'
          crossOrigin="anonymous" />
      )}
    </div>
  )
}

const editHandle = BaseDialog.createHandle<Payload>();

function Info({ id }: { id: number }) {
  const { data, isLoading } = useImage(id);
  const [copying, setCopying] = useState(false);

  const handleCopy = async () => {
    if (!data) return;

    setCopying(true);
    try {
      await copyImage(data.imageUrl);
      toast.success('Copied to clipboard');
    }
    catch (e) {
      toast.error(formatError(e));
    }
    finally {
      setCopying(false);
    }
  };

  return (
    <div className='w-full h-auto flex flex-col justify-center items-start gap-2'>

      {isLoading && (
        <Card className='my-2 w-full rounded-md py-4'>
          <CardContent className='px-4 flex flex-col gap-2'>
            <div className='flex flex-col items-center justify-center gap-2'>
              <Spinner /> Loading...
            </div>
          </CardContent>
        </Card>
      )}

      {data && (
        <Card className='my-2 w-full rounded-md py-4'>
          <CardContent className='px-4 flex flex-col gap-2'>
            <div>
              Image #{data.id}
            </div>
            <div>
              <Pilcrow size={12} className='inline' /> {data.text ? <>Text: {data.text}</> : <span className='italic'>(no text)</span>}
            </div>
            <div>
              <Hash size={12} className='inline' /> Tags: <Tags value={data.tags.map(it => it.name)} />
            </div>
            <div>
              <ShieldAlert size={12} className='inline' /> Rated as {data.rating}
            </div>
            <div>
              Created at {new Date(data.createdAt).toLocaleDateString()} by {data.userIdentifier.username}
            </div>
          </CardContent>
          <CardFooter className='flex flex-row gap-4 flex-wrap'>
            <Button variant='outline' onClick={() => {
              editHandle.openWithPayload({
                ...data,
                tags: data.tags.map(it => it.name)
              })
            }}>
              <Edit /> Edit
            </Button>

            <LikeButton id={data.id} />

            <Button
              variant='outline'
              disabled={copying}
              onClick={handleCopy}>
              {copying && <Spinner />}
              {copying || <Copy />} Copy
            </Button>

            <Button
              variant='outline'
              render={<a href={data.imageUrl} download={`image-${data.id}`} />}>
              <Download /> Download
            </Button>
          </CardFooter>
        </Card>
      )}
    </div>
  )
}

function Versions({ id }: { id: number }) {
  const { data, isLoading } = useImageVersions(id);

  return (
    <Card className='py-2 rounded-md'>

      {isLoading && (
        <div className='flex flex-col items-center justify-center gap-2'>
          <Spinner /> Loading...
        </div>
      )}

      {data && (
        <CardContent className='w-full px-2'>
          <Collapsible>
            <CollapsibleTrigger render={
              <Button variant='ghost' className='w-full'>
                {data.length} {pluralize('version', data.length)}
                <ChevronDownIcon className="ml-auto group-data-panel-open/button:rotate-180 transition-all" />
              </Button>
            } />

            <CollapsibleContent className='px-2 pb-2 flex flex-col items-start gap-2'>
              <ScrollArea className='w-full h-64'>
                <ul className='w-full min-w-0'>
                  {data.map(it => (
                    <li key={it.id} className='grid grid-cols-[max-content_minmax(0,1fr)] items-center not-first:border-t'>
                      <span className='whitespace-nowrap'>
                        Version {it.version}
                      </span>
                      <div className='min-w-0 flex flex-col px-2 py-1 gap-2 [overflow-wrap:anywhere]'>
                        <table className='w-full table-fixed border-collapse'>
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
                                {it.text ? <>Text: {it.text}</> : <span className='italic'>(no text)</span>}
                              </td>
                            </tr>
                            <tr>
                              <td className='py-1 align-top'>
                                <div className={`flex h-lh items-center ${it.tags.length > 0 ? 'box-content border-y border-transparent py-1' : ''}`}>
                                  <Hash size={12} />
                                </div>
                              </td>
                              <td className='py-1 align-top'>
                                <div className='flex flex-wrap items-center gap-1'>
                                  {it.tags.map(name => (
                                    <span key={name} className='min-w-0 max-w-full px-2 py-1 rounded-md bg-accent border'>
                                      {name}
                                    </span>
                                  ))}
                                  {it.tags.length === 0 && (
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
                                Rated as {it.rating}
                              </td>
                            </tr>
                          </tbody>
                        </table>
                        <span className='text-secondary-foreground text-sm'>
                          Created at {new Date(it.createdAt).toLocaleDateString()} by {it.userIdentifier.username}
                        </span>
                      </div>
                    </li>
                  ))}
                </ul>
              </ScrollArea>
            </CollapsibleContent>
          </Collapsible>
        </CardContent>
      )}
    </Card>
  )
}

export default function ImagePage() {
  const { id } = useParams();

  if (!id) {
    throw new Error();
  }

  return (
    <div className='w-full grid grid-cols-12 gap-4'>
      <EditDialog handle={editHandle} />
      <div className='col-span-12 md:col-span-4'>
        <Image id={Number(id)} />
      </div>
      <div className='col-span-12 min-w-0 md:col-span-8'>
        <Info id={Number(id)} />
        <Versions id={Number(id)} />
      </div>
    </div>
  )
}
