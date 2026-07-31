import { Download, Loader2, Search } from "lucide-react";
import { useMemo, useState } from "react";
import { PageShell } from "@/components/PageShell";
import { EmptyState, ErrorState, LoadingState } from "@/components/States";
import { cn } from "@/lib/cn";
import {
  useDeezerDownload,
  useDeezerJobs,
  useDeezerSearch,
  useDeezerStatus,
} from "@/lib/queries";
import type { DeezerResult, DeezerSearchType } from "@/lib/api";

const tabs: { id: DeezerSearchType; label: string }[] = [
  { id: "track", label: "Songs" },
  { id: "album", label: "Albums" },
  { id: "artist", label: "Artists" },
];

export function DeezerAdd() {
  const { data: status } = useDeezerStatus();
  const [query, setQuery] = useState("");
  const [type, setType] = useState<DeezerSearchType>("track");
  const [submitted, setSubmitted] = useState("");

  const { data, isPending, error } = useDeezerSearch(submitted, type);
  const download = useDeezerDownload();

  const hasActiveJobs = useMemo(() => {
    return status?.running;
  }, [status?.running]);
  const { data: jobsData } = useDeezerJobs(Boolean(hasActiveJobs));

  const activeJobs = jobsData?.jobs.filter((job) => job.state === "waiting" || job.state === "active") ?? [];
  const recentJobs = jobsData?.jobs.slice(-5).reverse() ?? [];

  function onSubmit(event: React.FormEvent) {
    event.preventDefault();
    setSubmitted(query.trim());
  }

  if (status && !status.configured) {
    return (
      <PageShell title="Add from Deezer" subtitle="Search and download music into your library">
        <EmptyState
          title="Deezer not configured"
          description={
            status.error ??
            "Set the DEEZER_ARL environment variable (your Deezer ARL cookie) and restart Spook."
          }
        />
      </PageShell>
    );
  }

  if (status && status.configured && !status.running) {
    return (
      <PageShell title="Add from Deezer">
        <ErrorState message={status.error ?? "Deezer subworker is not running."} />
      </PageShell>
    );
  }

  return (
    <PageShell
      title="Add from Deezer"
      subtitle="Downloads are saved to your music folder and indexed automatically."
    >
      <form onSubmit={onSubmit} className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center">
        <div className="relative flex-1">
          <Search className="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-tertiary" aria-hidden />
          <input
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search Deezer"
            aria-label="Search Deezer"
            className="w-full rounded-xl bg-fill py-2.5 pr-3 pl-10 text-[14px] outline-none focus:bg-fill-strong"
          />
        </div>
        <button
          type="submit"
          disabled={!query.trim()}
          className="rounded-xl bg-accent px-4 py-2.5 text-[13px] font-medium text-accent-content transition-opacity disabled:opacity-50"
        >
          Search
        </button>
      </form>

      <div className="mb-5 flex gap-1 rounded-lg bg-fill p-1">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            type="button"
            onClick={() => {
              setType(tab.id);
              if (submitted) setSubmitted(submitted);
            }}
            className={cn(
              "flex-1 rounded-md px-3 py-1.5 text-[13px] transition-colors",
              type === tab.id ? "bg-fill-strong text-content" : "text-secondary hover:text-content",
            )}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {!submitted ? (
        <EmptyState title="Search Deezer" description="Find songs and albums to add to your library." />
      ) : isPending ? (
        <LoadingState />
      ) : error ? (
        <ErrorState message={error.message} />
      ) : data && data.results.length === 0 ? (
        <EmptyState title="No results" description={`Nothing on Deezer matched “${submitted}”.`} />
      ) : (
        <div className="flex flex-col gap-2">
          {data?.results.map((result) => (
            <DeezerResultRow
              key={`${result.type}-${result.id}`}
              result={result}
              searchType={type}
              downloading={download.isPending}
              onDownload={(downloadType, musicId) => download.mutate({ type: downloadType, musicId })}
            />
          ))}
        </div>
      )}

      {(activeJobs.length > 0 || recentJobs.length > 0) && (
        <section className="mt-10">
          <h2 className="pb-3 text-[19px] font-bold">Downloads</h2>
          <div className="flex flex-col gap-2">
            {(activeJobs.length > 0 ? activeJobs : recentJobs).map((job) => (
              <div
                key={job.id}
                className="flex items-center gap-3 rounded-xl bg-fill px-3 py-2.5 text-[13px]"
              >
                {(job.state === "waiting" || job.state === "active") && (
                  <Loader2 className="h-4 w-4 shrink-0 animate-spin text-accent" aria-hidden />
                )}
                <div className="min-w-0 flex-1">
                  <div className="truncate">{job.description}</div>
                  <div className="text-[12px] text-secondary capitalize">{job.state.replaceAll("_", " ")}</div>
                </div>
                {job.progressMax > 0 && (
                  <span className="text-[12px] text-secondary">
                    {job.progress}/{job.progressMax}
                  </span>
                )}
              </div>
            ))}
          </div>
        </section>
      )}
    </PageShell>
  );
}

function DeezerResultRow({
  result,
  searchType,
  downloading,
  onDownload,
}: {
  result: DeezerResult;
  searchType: DeezerSearchType;
  downloading: boolean;
  onDownload: (type: "track" | "album", musicId: number) => void;
}) {
  const musicId = Number(result.id);
  const canDownloadTrack = searchType === "track" || result.type === "track";
  const canDownloadAlbum = searchType === "album" || (searchType === "track" && result.albumId);

  return (
    <div className="flex items-center gap-3 rounded-xl bg-fill px-3 py-2.5">
      {result.imageUrl ? (
        <img src={result.imageUrl} alt="" className="h-12 w-12 shrink-0 rounded-md object-cover shadow-art" />
      ) : (
        <div className="h-12 w-12 shrink-0 rounded-md bg-fill-strong" />
      )}
      <div className="min-w-0 flex-1">
        <div className="truncate text-[14px] font-medium">
          {result.title || result.album || result.artist || "Unknown"}
        </div>
        <div className="truncate text-[12px] text-secondary">
          {[result.artist, result.album].filter(Boolean).join(" · ")}
        </div>
      </div>
      <div className="flex shrink-0 gap-1.5">
        {canDownloadTrack && (
          <button
            type="button"
            disabled={downloading || !Number.isFinite(musicId)}
            onClick={() => onDownload("track", musicId)}
            className="flex items-center gap-1 rounded-lg bg-fill-strong px-2.5 py-1.5 text-[12px] transition-colors hover:bg-accent hover:text-accent-content disabled:opacity-50"
          >
            <Download className="h-3.5 w-3.5" aria-hidden />
            Song
          </button>
        )}
        {canDownloadAlbum && (
          <button
            type="button"
            disabled={downloading}
            onClick={() => onDownload("album", searchType === "album" ? musicId : Number(result.albumId))}
            className="flex items-center gap-1 rounded-lg bg-fill-strong px-2.5 py-1.5 text-[12px] transition-colors hover:bg-accent hover:text-accent-content disabled:opacity-50"
          >
            <Download className="h-3.5 w-3.5" aria-hidden />
            Album
          </button>
        )}
      </div>
    </div>
  );
}
