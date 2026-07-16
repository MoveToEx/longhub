import { Button } from "@/shared/components/ui/button";
import { Card, CardContent, CardFooter } from "@/shared/components/ui/card";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/shared/components/ui/collapsible";
import { Spinner } from "@/shared/components/ui/spinner";
import useImage from "@/features/images/hooks/use-image";
import useImageVersions from "@/features/images/hooks/use-image-versions";
import { Check, ChevronDownIcon, Edit, Hash, InfoIcon, Pilcrow, ShieldAlert, Undo2 } from "lucide-react";
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
import useAuth from "@/features/auth/hooks/use-auth";
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
import { Alert, AlertDescription } from "@/shared/components/ui/alert";
import { formatError } from "@/shared/lib/utils";


type Payload = {
  id: number,
  rating: string,
  tags: string[],
  text: string,
}

function RevertDialog({ handle }: { handle: BaseDialog.Handle<Payload> }) {
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
      {({ payload }) => {
        return (
          <DialogContent>
            <DialogHeader>
              <DialogTitle className='text-xl'>
                Confirm reversion
              </DialogTitle>
            </DialogHeader>

            <div className='flex flex-col items-start gap-2'>
              <div>
                You're about to revert image to the following state:
              </div>
              <ul className='flex flex-col gap-1 mb-4'>
                <li>
                  <Pilcrow size={12} className='inline' /> Text: {payload?.text}
                </li>
                <li>
                  <Hash size={12} className='inline' /> Tags:
                  <Tags value={payload?.tags ?? []} />
                </li>
                <li>
                  <ShieldAlert size={12} className='inline' /> Rating: {payload?.rating}
                </li>
              </ul>

              <Alert>
                <InfoIcon />
                <AlertDescription>
                  Reverting will create a new version with desired metadata, instead of deleting all versions after it.
                </AlertDescription>
              </Alert>
            </div>

            <DialogFooter>
              <Button
                variant='destructive'
                disabled={loading}
                onClick={() => payload && submit(payload)}>
                {loading && <Spinner />}
                {loading || <Check />} Confirm
              </Button>
              <DialogClose
                disabled={loading}
                render={
                  <Button variant='outline'>
                    Cancel
                  </Button>
                } />
            </DialogFooter>
          </DialogContent>
        )
      }}
    </Dialog>
  )
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
              <Pilcrow size={12} className='inline' /> Text: {data.text}
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
          </CardFooter>
        </Card>
      )}
    </div>
  )
}

const revertHandle = BaseDialog.createHandle<Payload>();

function Versions({ id }: { id: number }) {
  const { data, isLoading } = useImageVersions(id);
  const { data: user } = useAuth();

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
                <table className='w-full'>
                  <tbody>
                    {data.map(it => (
                      <tr key={it.id} className='not-first:border-t first:[&_button]:hidden'>
                        <td>
                          Version {it.version}
                        </td>
                        <td>
                          <div className='w-full flex flex-col px-2 py-1 gap-2'>
                            <span>
                              <Pilcrow size={12} className='inline' /> {it.text}
                            </span>
                            <span>
                              <Hash size={12} className='inline' /> {it.tags.map(name => (
                                <span key={name} className='px-2 py-1 rounded-md bg-accent border mr-1'>
                                  {name}
                                </span>
                              ))}
                              {it.tags.length === 0 && (
                                <span className='italic text-accent-foreground'>(No tags)</span>
                              )}
                            </span>
                            <span>
                              <ShieldAlert size={12} className='inline' /> Rated as {it.rating}
                            </span>
                            <span className='text-secondary-foreground text-sm'>
                              Created at {new Date(it.createdAt).toLocaleDateString()} by {it.userIdentifier.username}
                            </span>
                          </div>
                        </td>

                        <td>
                          <Button
                            variant='outline'
                            disabled={!user}
                            onClick={() => {
                              revertHandle.openWithPayload({
                                ...it,
                                id
                              });
                            }}>
                            <Undo2 /> Revert To
                          </Button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
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
      <RevertDialog handle={revertHandle} />
      <div className='col-span-12 md:col-span-4'>
        <Image id={Number(id)} />
      </div>
      <div className='col-span-12 md:col-span-8'>
        <Info id={Number(id)} />
        <Versions id={Number(id)} />
      </div>
    </div>
  )
}
