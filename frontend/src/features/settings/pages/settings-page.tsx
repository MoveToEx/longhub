import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/components/ui/tabs";
import { Braces, LockKeyhole, Settings } from "lucide-react";
import GeneralTab from "@/features/settings/components/general-tab";
import SecurityTab from "@/features/settings/components/security-tab";
import IntegrationTab from "@/features/settings/components/integration-tab";
import { RequiresLogin } from "@/shared/components/utils";

export default function SettingsPage() {
  return (
    <div className='w-full h-full flex flex-col mx-auto lg:max-w-4xl'>
      <RequiresLogin />
      <Tabs orientation='vertical' defaultValue='general'>
        <TabsList variant='line'>
          <TabsTrigger value='general'><Settings /> General</TabsTrigger>
          <TabsTrigger value='security'><LockKeyhole /> Security</TabsTrigger>
          <TabsTrigger value='integration'><Braces /> Integration</TabsTrigger>
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
      </Tabs>
    </div>
  )
}
