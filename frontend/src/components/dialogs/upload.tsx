import useAuth from '@/hooks/server/use-auth';
import useObjectUrl from '@/hooks/use-object-url';
import { zodResolver } from '@hookform/resolvers/zod';
import { useRef, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import z from 'zod';
import { Button } from "@/components/ui/button";
import { ChevronLeft, ChevronRight, FileTypeCorner, Send, Trash, UploadIcon, X } from "lucide-react";
import { Dialog, DialogClose, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import DragDrop from '../drag-drop';
import api from '@/lib/axios';
import type { Rating, SignResponse, Wrapped } from '@/lib/types';
import axios, { AxiosError } from 'axios';
import { Field, FieldError, FieldLabel } from '@/components/ui/field';
import { Input } from '@/components/ui/input';
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group';
import { Spinner } from '@/components/ui/spinner';
import { mutate } from '@/lib/swr';
import TagsInput from '../tags-input';

const uploadSchema = z.object({
  text: z.string().refine(val => val.indexOf(' ') === -1, 'Text should not contain spaces'),
  tags: z.string().array(),
  rating: z.enum(['none', 'moderate', 'violent'])
});

function UploadTrigger() {
  const { data } = useAuth();
  return (
    <DialogTrigger disabled={!data} render={
      <Button>
        <UploadIcon /> Upload
      </Button>
    } />
  )
}

export default function UploadDialog() {
  const [open, setOpen] = useState(false);
  const [files, setFiles] = useState<Blob[]>([]);
  const [selected, setSelected] = useState<number>(0);
  const [loading, setLoading] = useState(false);
  const formRef = useRef<HTMLFormElement | null>(null);
  const form = useForm<z.infer<typeof uploadSchema>>({
    resolver: zodResolver(uploadSchema),
    defaultValues: {
      text: '',
      tags: [],
      rating: 'none'
    }
  });

  const src = useObjectUrl(files?.[selected]);

  const submit = async (data: z.infer<typeof uploadSchema>, file: Blob) => {
    if (!file.type.startsWith('image')) {
      return;
    }

    setLoading(true);

    try {
      let response = await api.post<Wrapped<SignResponse>>('/image/sign', {
        mime: file.type
      });

      const { sessionId, url } = response.data.data;

      await axios.put(url, file, {
        headers: {
          'Content-Type': file.type
        }
      });

      response = await api.post('/image/ack', {
        sessionId,
        ...data,
      });

      form.reset();

      if (files.length === 1) {
        setOpen(false);
        mutate('image');
      }
      setFiles([...files.slice(undefined, selected), ...files.slice(selected + 1)]);
    }
    catch (e) {
      if (e instanceof AxiosError) {
        // 
      }
    }
    finally {
      setLoading(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={val => setOpen(val)}>
      <UploadTrigger />
      <DialogContent className='min-h-64 sm:min-w-2xl'>
        <DialogHeader>
          <DialogTitle className='text-lg'>
            Upload
          </DialogTitle>
        </DialogHeader>

        <div className='grid grid-cols-2 gap-4'>
          <div className='col-span-2 md:col-span-1'>
            <div className='w-full h-full flex flex-col justify-center'>
              {files.length === 0 && (
                <DragDrop
                  onChange={files => setFiles(files)}
                  accept='image/*'
                  multiple />
              )}

              {files.length !== 0 && src && (
                <div className='flex flex-col gap-2'>
                  <img src={src} className='w-full' alt='preview' crossOrigin='anonymous' />
                  <div className='flex flex-row justify-center items-center gap-2'>
                    <FileTypeCorner size={16} /> {files[selected]?.type}
                  </div>
                  <div className='flex flex-row justify-around items-center'>
                    <Button
                      variant='outline'
                      disabled={selected === 0}
                      onClick={() => {
                        setSelected(selected - 1);
                        form.reset();
                      }}>
                      <ChevronLeft />
                    </Button>

                    <span>
                      {selected + 1} / {files.length}
                    </span>

                    <Button
                      variant='outline'
                      disabled={selected + 1 === files.length}
                      onClick={() => {
                        setSelected(selected + 1);
                        form.reset();
                      }}>
                      <ChevronRight />
                    </Button>
                  </div>
                  <div className='flex flex-row justify-center gap-2'>
                    <Button
                      variant='outline'
                      onClick={() => {
                        if (selected + 1 === files.length) {
                          setSelected(selected - 1);
                        }
                        setFiles([...files.slice(undefined, selected), ...files.slice(selected + 1)]);
                        form.reset();
                      }}>
                      <X /> Skip
                    </Button>
                    <Button
                      variant='destructive'
                      onClick={() => {
                        setFiles([]);
                        setSelected(0);
                        form.reset();
                      }}>
                      <Trash /> Clear
                    </Button>
                  </div>
                </div>
              )}
            </div>
          </div>

          <div className='col-span-2 md:col-span-1'>

            <form ref={formRef} className='flex flex-col gap-4' onSubmit={form.handleSubmit(data => submit(data, files[0]))}>
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
            </form>
          </div>
        </div>

        <DialogFooter>
          <Button
            disabled={loading || files.length === 0}
            onClick={() => {
              formRef.current?.requestSubmit();
            }}>
            {loading || <Send />}
            {loading && <Spinner />}
            Submit
          </Button>
          <DialogClose render={
            <Button variant='outline' onClick={() => setOpen(false)}>
              <X /> Cancel
            </Button>
          } />
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
