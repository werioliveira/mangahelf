export default function ChapterLoading() {
  return (
    <div className="max-w-3xl mx-auto px-4 py-6 animate-pulse">
      <div className="h-12 rounded-xl bg-card mb-6" />
      <div className="space-y-4">
        <div className="w-full aspect-[2/3] rounded-xl bg-card" />
        <div className="w-full aspect-[2/3] rounded-xl bg-card" />
      </div>
    </div>
  );
}
