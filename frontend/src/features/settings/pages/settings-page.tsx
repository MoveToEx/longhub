import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/components/ui/tabs";
import { KeyRound, LockKeyhole, Settings, Webhook } from "lucide-react";
import GeneralTab from "@/features/settings/components/general-tab";
import SecurityTab from "@/features/settings/components/security-tab";
import IntegrationTab from "@/features/settings/components/integration-tab";
import WebhooksTab from "@/features/settings/components/webhooks-tab";
import { RequiresLogin } from "@/shared/components/utils";
import { Separator } from "@/shared/components/ui/separator";

export default function SettingsPage() {
  return (
    <div className='w-full h-full flex flex-col mx-auto lg:max-w-4xl'>
      <RequiresLogin />
      <Tabs orientation='vertical' defaultValue='general'>
        <TabsList variant='line' className='w-48'>
          <TabsTrigger value='general'><Settings /> General</TabsTrigger>
          <TabsTrigger value='security'><LockKeyhole /> Security</TabsTrigger>
          <Separator className='mt-2' />
          <p className='text-xs w-full my-2 select-none'>Integration</p>
          <TabsTrigger value='integration'><KeyRound /> Appkeys</TabsTrigger>
          <TabsTrigger value='webhooks'><Webhook /> Webhooks</TabsTrigger>
        </TabsList>
        <TabsContent value='general'>
          <GeneralTab />
        </TabsContent>
        <TabsContent value='security'>
          <SecurityTab />
        </TabsContent>
        <TabsContent value='integration'>
          <IntegrationTab />
        </TabsContent>
        <TabsContent value='webhooks'>
          <WebhooksTab />
        </TabsContent>
      </Tabs>
    </div>
  )
}
