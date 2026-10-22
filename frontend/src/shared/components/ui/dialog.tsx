"use client"

import * as React from "react"
import { Dialog as DialogPrimitive } from "@base-ui/react/dialog"

import { cn } from "@/shared/lib/utils"
import { Button } from "@/shared/components/ui/button"
import { XIcon } from "lucide-react"

function Dialog<T = unknown>({ ...props }: DialogPrimitive.Root.Props<T>) {
  return <DialogPrimitive.Root data-slot="dialog" {...props} />
}

function DialogTrigger({ ...props }: DialogPrimitive.Trigger.Props) {
  return <DialogPrimitive.Trigger data-slot="dialog-trigger" {...props} />
}

function DialogPortal({ ...props }: DialogPrimitive.Portal.Props) {
  return <DialogPrimitive.Portal data-slot="dialog-portal" {...props} />
}

function DialogClose({ ...props }: DialogPrimitive.Close.Props) {
  return <DialogPrimitive.Close data-slot="dialog-close" {...props} />
}

function DialogOverlay({
  className,
  ...props
}: DialogPrimitive.Backdrop.Props) {
  return (
    <DialogPrimitive.Backdrop
      data-slot="dialog-overlay"
      className={cn("bg-black/50 fixed inset-0 isolate z-50 opacity-100 transition-opacity duration-100 data-starting-style:opacity-0 data-ending-style:opacity-0 motion-reduce:transition-none", className)}
      {...props}
    />
  )
}

function DialogContent({
  className,
  children,
  showCloseButton = true,
  mobileDrawer = false,
  ...props
}: DialogPrimitive.Popup.Props & {
  showCloseButton?: boolean
  mobileDrawer?: boolean
}) {
  const dragStartY = React.useRef<number | null>(null)

  return (
    <DialogPortal>
      <DialogOverlay forceRender={mobileDrawer} className={mobileDrawer ? "max-md:duration-300" : undefined} />
      <DialogPrimitive.Popup
        data-slot="dialog-content"
        className={cn(
          "bg-background data-open:animate-in data-closed:animate-out ring-foreground/10 fixed z-50 grid w-full gap-6 p-6 text-sm ring-1 outline-none",
          mobileDrawer
            ? "inset-x-0 bottom-0 max-h-[calc(100dvh-1rem)] overflow-y-auto overscroll-contain rounded-none pt-10 pb-[max(1.5rem,env(safe-area-inset-bottom))] duration-300 data-open:slide-in-from-bottom data-closed:slide-out-to-bottom motion-reduce:animate-none md:inset-x-auto md:bottom-auto md:top-1/2 md:left-1/2 md:max-w-md md:-translate-x-1/2 md:-translate-y-1/2 md:rounded-xl md:pt-6 md:pb-6 md:duration-100 md:data-open:slide-in-from-bottom-0 md:data-closed:slide-out-to-bottom-0 md:data-open:fade-in-0 md:data-closed:fade-out-0 md:data-open:zoom-in-95 md:data-closed:zoom-out-95"
            : "top-1/2 left-1/2 max-w-[calc(100%-2rem)] -translate-x-1/2 -translate-y-1/2 rounded-xl duration-100 data-open:fade-in-0 data-closed:fade-out-0 data-open:zoom-in-95 data-closed:zoom-out-95 sm:max-w-md",
          className
        )}
        {...props}
      >
        {mobileDrawer && (
          <DialogPrimitive.Close
            data-slot="drawer-handle"
            aria-label="Drag down to close drawer"
            className="absolute inset-x-0 top-0 flex h-10 touch-none cursor-grab items-center justify-center outline-none select-none focus-visible:ring-2 focus-visible:ring-ring active:cursor-grabbing md:hidden"
            onPointerDown={(event) => {
              if (event.button !== 0) return
              dragStartY.current = event.clientY
              event.currentTarget.setPointerCapture(event.pointerId)
            }}
            onPointerUp={(event) => {
              const startY = dragStartY.current
              dragStartY.current = null
              if (startY !== null && event.clientY - startY >= 48) {
                event.currentTarget.click()
              }
            }}
            onPointerCancel={() => { dragStartY.current = null }}
            onClick={(event) => {
              // Pointer dismissal requires a downward drag; keyboard activation still closes.
              if (event.detail !== 0) event.preventBaseUIHandler()
            }}
          >
            <span aria-hidden="true" className="bg-muted-foreground/30 h-1 w-12 rounded-full" />
          </DialogPrimitive.Close>
        )}
        {children}
        {showCloseButton && (
          <DialogPrimitive.Close
            data-slot="dialog-close"
            render={
              <Button
                variant="ghost"
                className={cn("absolute top-4 right-4", mobileDrawer && "hidden md:inline-flex")}
                size="icon-sm"
              />
            }
          >
            <XIcon
            />
            <span className="sr-only">Close</span>
          </DialogPrimitive.Close>
        )}
      </DialogPrimitive.Popup>
    </DialogPortal>
  )
}

function DialogHeader({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="dialog-header"
      className={cn("gap-2 flex flex-col", className)}
      {...props}
    />
  )
}

function DialogFooter({
  className,
  showCloseButton = false,
  children,
  ...props
}: React.ComponentProps<"div"> & {
  showCloseButton?: boolean
}) {
  return (
    <div
      data-slot="dialog-footer"
      className={cn(
        "gap-2 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end",
        className
      )}
      {...props}
    >
      {children}
      {showCloseButton && (
        <DialogPrimitive.Close render={<Button variant="outline" />}>
          Close
        </DialogPrimitive.Close>
      )}
    </div>
  )
}

function DialogTitle({ className, ...props }: DialogPrimitive.Title.Props) {
  return (
    <DialogPrimitive.Title
      data-slot="dialog-title"
      className={cn("leading-none font-medium", className)}
      {...props}
    />
  )
}

function DialogDescription({
  className,
  ...props
}: DialogPrimitive.Description.Props) {
  return (
    <DialogPrimitive.Description
      data-slot="dialog-description"
      className={cn("text-muted-foreground *:[a]:hover:text-foreground text-sm *:[a]:underline *:[a]:underline-offset-3", className)}
      {...props}
    />
  )
}

export {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogOverlay,
  DialogPortal,
  DialogTitle,
  DialogTrigger,
}
