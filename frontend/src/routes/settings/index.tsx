import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Braces, LockKeyhole, Settings } from "lucide-react";
import GeneralTab from "./general";
import SecurityTab from "./security";
import IntegrationTab from "./integration";
import { RequiresLogin } from "@/components/utils";

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