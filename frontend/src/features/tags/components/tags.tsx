export default function Tags({ value }: { value: string[] }) {
  return value.map(it => (
    <span key={it} className='min-w-0 max-w-full px-2 py-1 rounded-md bg-accent border'>
      {it}
    </span>
  ))
}
