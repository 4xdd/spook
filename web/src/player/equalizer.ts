import { EQ_BANDS, defaultEqSettings, type EqSettings } from "@/lib/eq";

type AudioContextConstructor = new () => AudioContext;

function audioContextConstructor(): AudioContextConstructor | null {
  if (typeof window === "undefined") return null;
  const legacy = (window as { webkitAudioContext?: AudioContextConstructor }).webkitAudioContext;
  return window.AudioContext ?? legacy ?? null;
}

/** Band edges in Hz, split where a mix separates rather than evenly. */
const LEVEL_BANDS = {
  low: [30, 250],
  mid: [250, 2000],
  high: [2000, 7000],
} as const;

export interface AudioLevels {
  low: number;
  mid: number;
  high: number;
}

/**
 * Routes the player's <audio> element through a chain of biquad filters and an
 * analyser tap.
 *
 * The graph is built lazily: constructing an AudioContext before the user has
 * interacted leaves it suspended, so we wait for the first play, for the
 * equaliser to be switched on, or for a view that wants levels to ask (all are
 * gestures). Once built it stays, and bypass is expressed as 0dB on every band,
 * which is mathematically transparent, rather than by rewiring the graph
 * mid-playback.
 */
class Equalizer {
  private element: HTMLAudioElement | null = null;
  private context: AudioContext | null = null;
  private source: MediaElementAudioSourceNode | null = null;
  private filters: BiquadFilterNode[] = [];
  private analyser: AnalyserNode | null = null;
  private spectrum: Uint8Array<ArrayBuffer> | null = null;
  private settings: EqSettings = defaultEqSettings();
  private unavailable = false;

  get supported(): boolean {
    return audioContextConstructor() !== null && !this.unavailable;
  }

  /**
   * Builds the graph if it is not up yet, for callers that want to read levels.
   * Only worth calling from a user gesture, or the context stays suspended.
   */
  ensureAnalyser(): boolean {
    this.build();
    void this.context?.resume();
    return this.analyser !== null;
  }

  /** Energy per band, 0 to 1, or null when there is no graph to read. */
  levels(): AudioLevels | null {
    const analyser = this.analyser;
    const context = this.context;
    const spectrum = this.spectrum;
    if (!analyser || !context || !spectrum) return null;

    analyser.getByteFrequencyData(spectrum);
    const perBin = context.sampleRate / analyser.fftSize;

    const read = ([from, to]: readonly [number, number]) => {
      const first = Math.max(1, Math.floor(from / perBin));
      const last = Math.min(spectrum.length - 1, Math.ceil(to / perBin));
      let sum = 0;
      for (let i = first; i <= last; i += 1) sum += spectrum[i];
      return sum / (last - first + 1) / 255;
    };

    return {
      low: read(LEVEL_BANDS.low),
      mid: read(LEVEL_BANDS.mid),
      high: read(LEVEL_BANDS.high),
    };
  }

  attach(audio: HTMLAudioElement) {
    if (this.element === audio) return;
    this.teardown();
    this.element = audio;
    audio.addEventListener("play", this.onPlay);
  }

  apply(settings: EqSettings) {
    this.settings = settings;
    if (settings.enabled) this.build();
    this.push();
  }

  private onPlay = () => {
    if (this.settings.enabled) this.build();
    // Browsers suspend contexts in backgrounded tabs; silence would outlast it.
    void this.context?.resume();
  };

  private buildAnalyser(context: AudioContext): AnalyserNode {
    const analyser = context.createAnalyser();
    analyser.fftSize = 1024;
    // Enough smoothing that the bars breathe instead of jittering.
    analyser.smoothingTimeConstant = 0.75;
    this.spectrum = new Uint8Array(new ArrayBuffer(analyser.frequencyBinCount));
    return analyser;
  }

  private build() {
    if (this.context || this.unavailable) return;

    const audio = this.element;
    const AudioContextClass = audioContextConstructor();
    if (!audio || !AudioContextClass) return;

    try {
      const context = new AudioContextClass();
      const source = context.createMediaElementSource(audio);

      let node: AudioNode = source;
      this.filters = EQ_BANDS.map((band) => {
        const filter = context.createBiquadFilter();
        filter.type = band.type;
        filter.frequency.value = band.frequency;
        filter.Q.value = band.q;
        filter.gain.value = 0;
        node.connect(filter);
        node = filter;
        return filter;
      });

      // The tap sits last so it hears what the speakers hear, filters included.
      const analyser = this.buildAnalyser(context);
      node.connect(analyser);
      analyser.connect(context.destination);

      this.context = context;
      this.source = source;
      this.analyser = analyser;
      void context.resume();
      this.push();
    } catch {
      // Without Web Audio the element still plays; only the filters are lost.
      this.unavailable = true;
      this.filters = [];
      this.analyser = null;
      this.spectrum = null;
    }
  }

  private push() {
    const context = this.context;
    if (!context) return;

    const now = context.currentTime;
    this.filters.forEach((filter, index) => {
      const gain = this.settings.enabled ? (this.settings.gains[index] ?? 0) : 0;
      // Ramp rather than jump: stepping a filter gain clicks audibly, and
      // dragging a band would otherwise fire a step per frame.
      filter.gain.setTargetAtTime(gain, now, 0.015);
    });
  }

  private teardown() {
    this.element?.removeEventListener("play", this.onPlay);
    this.source?.disconnect();
    this.filters.forEach((filter) => filter.disconnect());
    this.analyser?.disconnect();
    void this.context?.close().catch(() => {});
    this.element = null;
    this.source = null;
    this.filters = [];
    this.analyser = null;
    this.spectrum = null;
    this.context = null;
  }
}

export const equalizer = new Equalizer();
