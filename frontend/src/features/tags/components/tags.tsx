export default function Tags({ value }: { value: string[] }) {
  return value.map(it => (
    <span key={it} className='px-2 py-1 rounded-md bg-accent border mr-1'>
      {it}
    </span>
  ))
}