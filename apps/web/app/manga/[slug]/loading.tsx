export default function SeriesLoading() {
  return (
    <div className="max-w-[1400px] mx-auto px-6 py-10 md:py-14 animate-pulse">
      <div className="flex flex-col md:flex-row gap-8 md:gap-12 items-center md:items-start">
        <div className="w-52 md:w-64 h-72 md:h-96 rounded-2xl bg-card" />
        <div className="flex-1 space-y-4 w-full">
          <div className="h-5 w-40 rounded-full bg-card" />
          <div className="h-10 w-2/3 rounded-xl bg-card" />
          <div className="h-4 w-1/3 rounded-full bg-card" />
          <div className="h-20 w-full rounded-xl bg-card" />
        </div>
      </div>
      <div className="mt-10 space-y-2 max-w-3xl">
        {Array.from({ length: 8 }, (_, i) => (
          <div key={i} className="h-12 rounded-lg bg-card" />
        ))}
      </div>
    </div>
  );
}
