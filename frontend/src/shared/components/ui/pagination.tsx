import * as React from "react"

import { cn } from "@/shared/lib/utils"
import { Button } from "@/shared/components/ui/button"
import {
  ChevronsLeftIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ChevronsRightIcon,
  MoreHorizontalIcon,
} from "lucide-react"

type PaginationItemType = "page" | "first" | "last" | "next" | "previous" | "start-ellipsis" | "end-ellipsis"

type PaginationRenderItemParams = {
  type: PaginationItemType
  page: number | null
  selected: boolean
  disabled: boolean
  "aria-label": string
  onClick: React.MouseEventHandler<HTMLAnchorElement>
}

type PaginationProps = Omit<React.ComponentProps<"nav">, "onChange"> & {
  count?: number
  page?: number
  defaultPage?: number
  onChange?: (event: React.MouseEvent<HTMLAnchorElement>, page: number) => void
  boundaryCount?: number
  siblingCount?: number
  disabled?: boolean
  hideNextButton?: boolean
  hidePrevButton?: boolean
  showFirstButton?: boolean
  showLastButton?: boolean
  getItemAriaLabel?: (type: PaginationItemType, page: number | null, selected: boolean) => string
  renderItem?: (item: PaginationRenderItemParams) => React.ReactNode
}

const defaultGetItemAriaLabel = (
  type: PaginationItemType,
  page: number | null,
  selected: boolean,
) => {
  if (type === "page") {
    return selected ? `Page ${page}` : `Go to page ${page}`
  }

  return `Go to ${type.replace("-", " ")} page`
}

const range = (start: number, end: number) => {
  const length = end - start + 1

  return length > 0 ? Array.from({ length }, (_, index) => start + index) : []
}

function usePaginationItems({
  count = 1,
  page = 1,
  boundaryCount = 1,
  siblingCount = 1,
  hideNextButton,
  hidePrevButton,
  showFirstButton,
  showLastButton,
}: Pick<
  PaginationProps,
  | "count"
  | "page"
  | "boundaryCount"
  | "siblingCount"
  | "hideNextButton"
  | "hidePrevButton"
  | "showFirstButton"
  | "showLastButton"
>) {
  const totalPages = Math.max(1, count)
  const currentPage = Math.min(Math.max(1, page), totalPages)
  const startPages = range(1, Math.min(boundaryCount, totalPages))
  const endPages = range(
    Math.max(totalPages - boundaryCount + 1, boundaryCount + 1),
    totalPages,
  )
  const siblingsStart = Math.max(
    Math.min(
      currentPage - siblingCount,
      totalPages - boundaryCount - siblingCount * 2 - 1,
    ),
    boundaryCount + 2,
  )
  const siblingsEnd = Math.min(
    Math.max(
      currentPage + siblingCount,
      boundaryCount + siblingCount * 2 + 2,
    ),
    endPages.length > 0 ? endPages[0] - 2 : totalPages - 1,
  )

  const items: (number | PaginationItemType)[] = [
    ...(showFirstButton ? ["first" as const] : []),
    ...(hidePrevButton ? [] : ["previous" as const]),
    ...startPages,
    ...(siblingsStart > boundaryCount + 2
      ? ["start-ellipsis" as const]
      : boundaryCount + 1 < totalPages - boundaryCount
        ? [boundaryCount + 1]
        : []),
    ...range(siblingsStart, siblingsEnd),
    ...(siblingsEnd < totalPages - boundaryCount - 1
      ? ["end-ellipsis" as const]
      : totalPages - boundaryCount > boundaryCount
        ? [totalPages - boundaryCount]
        : []),
    ...endPages,
    ...(hideNextButton ? [] : ["next" as const]),
    ...(showLastButton ? ["last" as const] : []),
  ]

  return {
    currentPage,
    totalPages,
    items,
  }
}

function Pagination({
  className,
  children,
  count,
  page,
  defaultPage = 1,
  onChange,
  boundaryCount = 1,
  siblingCount = 1,
  disabled = false,
  hideNextButton = false,
  hidePrevButton = false,
  showFirstButton = false,
  showLastButton = false,
  getItemAriaLabel = defaultGetItemAriaLabel,
  renderItem,
  ...props
}: PaginationProps) {
  const isConveniencePagination = count !== undefined
  const [uncontrolledPage, setUncontrolledPage] = React.useState(defaultPage)
  const { currentPage, totalPages, items } = usePaginationItems({
    count,
    page: page ?? uncontrolledPage,
    boundaryCount,
    siblingCount,
    hideNextButton,
    hidePrevButton,
    showFirstButton,
    showLastButton,
  })

  const handleChange = (event: React.MouseEvent<HTMLAnchorElement>, nextPage: number) => {
    if (disabled || nextPage === currentPage || nextPage < 1 || nextPage > totalPages) {
      event.preventDefault()
      return
    }

    if (page === undefined) {
      setUncontrolledPage(nextPage)
    }

    onChange?.(event, nextPage)
  }

  return (
    <nav
      role="navigation"
      aria-label="pagination"
      data-slot="pagination"
      className={cn(
        "mx-auto flex w-full justify-center",
        className
      )}
      {...props}
    >
      {isConveniencePagination ? (
        <PaginationContent>
          {items.map((item, index) => {
            const type: PaginationItemType = typeof item === "number" ? "page" : item
            const itemPage = typeof item === "number"
              ? item
              : item === "first"
                ? 1
                : item === "previous"
                  ? currentPage - 1
                  : item === "next"
                    ? currentPage + 1
                    : item === "last"
                      ? totalPages
                      : null
            const isEllipsis = type === "start-ellipsis" || type === "end-ellipsis"
            const selected = type === "page" && itemPage === currentPage
            const itemDisabled = disabled
              || isEllipsis
              || itemPage === null
              || (type === "previous" && currentPage === 1)
              || (type === "first" && currentPage === 1)
              || (type === "next" && currentPage === totalPages)
              || (type === "last" && currentPage === totalPages)
            const renderParams: PaginationRenderItemParams = {
              type,
              page: itemPage,
              selected,
              disabled: itemDisabled,
              "aria-label": getItemAriaLabel(type, itemPage, selected),
              onClick: event => {
                if (itemPage === null) {
                  event.preventDefault()
                  return
                }

                handleChange(event, itemPage)
              },
            }

            return (
              <PaginationItem key={`${type}-${itemPage ?? index}`}>
                {renderItem ? renderItem(renderParams) : <PaginationControl {...renderParams} />}
              </PaginationItem>
            )
          })}
        </PaginationContent>
      ) : children}
    </nav>
  )
}

function PaginationContent({
  className,
  ...props
}: React.ComponentProps<"ul">) {
  return (
    <ul
      data-slot="pagination-content"
      className={cn("gap-1 flex items-center", className)}
      {...props}
    />
  )
}

function PaginationItem({ ...props }: React.ComponentProps<"li">) {
  return <li data-slot="pagination-item" {...props} />
}

type PaginationLinkProps = {
  isActive?: boolean
} & Pick<React.ComponentProps<typeof Button>, "size" | "disabled"> &
  React.ComponentProps<"a">

function PaginationLink({
  className,
  isActive,
  size = "icon",
  disabled,
  ...props
}: PaginationLinkProps) {
  return (
    <Button
      variant={isActive ? "outline" : "ghost"}
      size={size}
      disabled={disabled}
      className={cn(className)}
      nativeButton={false}
      render={
        <a
          aria-current={isActive ? "page" : undefined}
          data-slot="pagination-link"
          data-active={isActive}
          {...props}
        />
      }
    />
  )
}

function PaginationPrevious({
  className,
  ...props
}: React.ComponentProps<typeof PaginationLink>) {
  return (
    <PaginationLink
      aria-label="Go to previous page"
      size="default"
      className={cn("pl-2!", className)}
      {...props}
    >
      <ChevronLeftIcon data-icon="inline-start" />
      <span className="hidden sm:block">
        Previous
      </span>
    </PaginationLink>
  )
}

function PaginationNext({
  className,
  ...props
}: React.ComponentProps<typeof PaginationLink>) {
  return (
    <PaginationLink
      aria-label="Go to next page"
      size="default"
      className={cn("pr-2!", className)}
      {...props}
    >
      <span className="hidden sm:block">Next</span>
      <ChevronRightIcon data-icon="inline-end" />
    </PaginationLink>
  )
}

function PaginationEllipsis({
  className,
  ...props
}: React.ComponentProps<"span">) {
  return (
    <span
      aria-hidden
      data-slot="pagination-ellipsis"
      className={cn(
        "size-9 items-center justify-center [&_svg:not([class*='size-'])]:size-4 flex items-center justify-center",
        className
      )}
      {...props}
    >
      <MoreHorizontalIcon
      />
      <span className="sr-only">More pages</span>
    </span>
  )
}

function PaginationControl({
  type,
  page,
  selected,
  disabled,
  ...props
}: PaginationRenderItemParams) {
  if (type === "start-ellipsis" || type === "end-ellipsis") {
    return <PaginationEllipsis />
  }

  if (type === "previous") {
    return <PaginationPrevious disabled={disabled} {...props} />
  }

  if (type === "next") {
    return <PaginationNext disabled={disabled} {...props} />
  }

  if (type === "first") {
    return (
      <PaginationLink disabled={disabled} size="default" {...props}>
        <ChevronsLeftIcon data-icon="inline-start" />
        <span className="hidden sm:block">First</span>
      </PaginationLink>
    )
  }

  if (type === "last") {
    return (
      <PaginationLink disabled={disabled} size="default" {...props}>
        <span className="hidden sm:block">Last</span>
        <ChevronsRightIcon data-icon="inline-end" />
      </PaginationLink>
    )
  }

  return (
    <PaginationLink isActive={selected} disabled={disabled} {...props}>
      {page}
    </PaginationLink>
  )
}

export {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
}
