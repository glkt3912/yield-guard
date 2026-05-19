import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cachedFetch, invalidateCache } from "../apiCache";

beforeEach(() => {
  invalidateCache();
  vi.useFakeTimers();
});

afterEach(() => {
  invalidateCache();
  vi.useRealTimers();
});

describe("cachedFetch", () => {
  it("returns cached value without calling fetcher again", async () => {
    const fetcher = vi.fn().mockResolvedValue({ value: 42 });

    const r1 = await cachedFetch("key1", 60_000, fetcher);
    const r2 = await cachedFetch("key1", 60_000, fetcher);

    expect(r1).toEqual({ value: 42 });
    expect(r2).toEqual({ value: 42 });
    expect(fetcher).toHaveBeenCalledTimes(1);
  });

  it("calls fetcher again after TTL expires", async () => {
    const fetcher = vi.fn().mockResolvedValue("data");
    const TTL = 5_000;

    await cachedFetch("key2", TTL, fetcher);
    vi.advanceTimersByTime(TTL + 1);
    await cachedFetch("key2", TTL, fetcher);

    expect(fetcher).toHaveBeenCalledTimes(2);
  });

  it("deduplicates concurrent in-flight requests", async () => {
    let resolveFirst!: (v: string) => void;
    const inflight = new Promise<string>((r) => (resolveFirst = r));
    const fetcher = vi.fn().mockReturnValue(inflight);

    const p1 = cachedFetch("key3", 60_000, fetcher);
    const p2 = cachedFetch("key3", 60_000, fetcher);

    resolveFirst("shared");
    const [r1, r2] = await Promise.all([p1, p2]);

    expect(r1).toBe("shared");
    expect(r2).toBe("shared");
    expect(fetcher).toHaveBeenCalledTimes(1);
  });

  it("does not cache errors and retries on next call", async () => {
    const fetcher = vi
      .fn()
      .mockRejectedValueOnce(new Error("network"))
      .mockResolvedValueOnce("ok");

    await expect(cachedFetch("key4", 60_000, fetcher)).rejects.toThrow("network");
    const result = await cachedFetch("key4", 60_000, fetcher);

    expect(result).toBe("ok");
    expect(fetcher).toHaveBeenCalledTimes(2);
  });

  it("caches null values", async () => {
    const fetcher = vi.fn().mockResolvedValue(null);

    const r1 = await cachedFetch("key5", 60_000, fetcher);
    const r2 = await cachedFetch("key5", 60_000, fetcher);

    expect(r1).toBeNull();
    expect(r2).toBeNull();
    expect(fetcher).toHaveBeenCalledTimes(1);
  });

  it("treats different keys independently", async () => {
    const fetcherA = vi.fn().mockResolvedValue("a");
    const fetcherB = vi.fn().mockResolvedValue("b");

    const a = await cachedFetch("keyA", 60_000, fetcherA);
    const b = await cachedFetch("keyB", 60_000, fetcherB);

    expect(a).toBe("a");
    expect(b).toBe("b");
    expect(fetcherA).toHaveBeenCalledTimes(1);
    expect(fetcherB).toHaveBeenCalledTimes(1);
  });
});

describe("invalidateCache", () => {
  it("clears all entries when called without prefix", async () => {
    const fetcher = vi.fn().mockResolvedValue("x");

    await cachedFetch("a:1", 60_000, fetcher);
    await cachedFetch("b:1", 60_000, fetcher);
    invalidateCache();
    await cachedFetch("a:1", 60_000, fetcher);
    await cachedFetch("b:1", 60_000, fetcher);

    expect(fetcher).toHaveBeenCalledTimes(4);
  });

  it("clears only prefix-matching entries", async () => {
    const fetcherA = vi.fn().mockResolvedValue("a");
    const fetcherB = vi.fn().mockResolvedValue("b");

    await cachedFetch("municipalities:13", 60_000, fetcherA);
    await cachedFetch("rentStats:13::", 60_000, fetcherB);

    invalidateCache("municipalities");

    await cachedFetch("municipalities:13", 60_000, fetcherA);
    await cachedFetch("rentStats:13::", 60_000, fetcherB);

    expect(fetcherA).toHaveBeenCalledTimes(2);
    expect(fetcherB).toHaveBeenCalledTimes(1);
  });
});
