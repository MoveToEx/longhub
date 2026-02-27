import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Separator } from "@/components/ui/separator";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import useAuth from "@/hooks/server/use-auth";
import api from "@/lib/axios";
import { zodResolver } from "@hookform/resolvers/zod";
import { AxiosError } from "axios";
import { Save } from "lucide-react";
import { useEffect, useMemo, useState } from "react"
import { Controller, useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from 'zod';

const schema = z.object({
  hideNSFW: z.boolean(),
  hideViolent: z.boolean(),
});

const omittedSchema = z.object({
  hideNSFW: z.boolean().default(false),
  hideViolent: z.boolean().default(false),
});


export default function GeneralTab() {
  const [loading, setLoading] = useState(false);
  const { data: user, isLoading, mutate } = useAuth();

  const form = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: useMemo(() => {
      if (user?.preference) {
        const { data, error } = omittedSchema.safeParse(user?.preference);
        if (error) {
          return {
            hideNSFW: false,
            hideViolent: false,
          }
        }
        return data;
      }
      return {
        hideNSFW: false,
        hideViolent: false,
      }
    }, [user?.preference]),
    disabled: loading || isLoading,
  });

  useEffect(() => {
    if (!user) return;

    const { data, error } = omittedSchema.safeParse(user?.preference);
    if (error) {
      return;
    }

    form.reset(data);

  }, [user, form]);

  const onSubmit = async (data: z.infer<typeof schema>) => {
    setLoading(true);
    try {
      await api.patch('/user/preferences', data);
      await mutate();
      toast.success('Saved');
    }
    catch (e) {
      if (e instanceof AxiosError) {
        toast.error(e.response?.data.error);
      }
    }
    finally {
      setLoading(false);
    }
  }

  return (
    <div className='w-full flex flex-col gap-2 items-center'>
      {isLoading && (
        <div className='flex flex-row items-center gap-4 h-32'>
          <Spinner className='inline' /> Loading
        </div>
      )}

      {user && (
        <form onSubmit={form.handleSubmit(onSubmit)} className='w-full md:w-1/2'>
          <span className='text-lg'>Content Preference</span>

          <FieldGroup className='mt-4'>
            <Controller
              name='hideNSFW'
              control={form.control}
              render={({ field, fieldState }) => (
                <Field data-invalid={fieldState.invalid} className='flex flex-row gap-2'>
                  <Switch
                    {...field}
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    value={undefined}
                    id="form-preference-hide-nsfw"
                  />
                  {fieldState.invalid && (
                    <FieldError errors={[fieldState.error]} />
                  )}
                  <FieldLabel htmlFor="form-preference-hide-nsfw">
                    Hide NSFW images
                    {fieldState.isDirty && <span>*</span>}
                  </FieldLabel>
                </Field>
              )} />
            <Controller
              name='hideViolent'
              control={form.control}
              render={({ field, fieldState }) => (
                <Field data-invalid={fieldState.invalid} className='flex flex-row gap-2'>
                  <Switch
                    {...field}
                    onCheckedChange={field.onChange}
                    checked={field.value}
                    value={undefined}
                    id="form-preference-hide-violent"
                  />
                  <FieldLabel htmlFor="form-preference-hide-violent">
                    Hide violent images
                    {fieldState.isDirty && <span>*</span>}
                  </FieldLabel>
                  {fieldState.invalid && (
                    <FieldError errors={[fieldState.error]} />
                  )}
                </Field>
              )} />
          </FieldGroup>

          <Separator className='my-4' />

          <div className='flex flex-row justify-end items-center gap-4'>
            <Button disabled={isLoading || loading} type='submit'>
              {loading && <Spinner />}
              {loading || <Save />}
              Save
            </Button>
            <Button variant='outline'>
              Cancel
            </Button>
          </div>
        </form>
      )}
    </div>
  )
}