"use client"

import type { ComponentProps } from "react"

import { Button } from "@/shared/components/ui/button"
import { DialogClose, DialogContent } from "@/shared/components/ui/dialog"
import { cn } from "@/shared/lib/utils"

/** Keeps one dialog mounted while adapting its presentation at the md breakpoint. */
function ResponsiveDialogContent(props: Omit<ComponentProps<typeof DialogContent>, "mobileDrawer">) {
  return <DialogContent mobileDrawer {...props} />
}

function ResponsiveDialogCancel({
  className,
  children = "Cancel",
  render = <Button variant="outline" />,
  ...props
}: ComponentProps<typeof DialogClose>) {
  return (
    <DialogClose
      className={cn("hidden md:inline-flex", className)}
      render={render}
      {...props}
    >
      {children}
    </DialogClose>
  )
}

export { ResponsiveDialogCancel, ResponsiveDialogContent }
