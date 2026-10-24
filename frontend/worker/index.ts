interface Env {
  ASSETS: Fetcher;
  BACKEND_URL: string;
  APP_INTERNAL_API_KEY?: string;
}

// 共有 URL のクエリから OG 用の説明文を作る（旧 Next.js generateMetadata の移植）
// クイックモード: totalPrice、フルモード: landPrice + buildingCost を合算
export function shareDescription(params: URLSearchParams): string | null {
  const quickPrice = Number(params.get("totalPrice"));
  const fullPrice =
    (Number(params.get("landPrice")) || 0) + (Number(params.get("buildingCost")) || 0);
  const resolvedPrice = quickPrice > 0 ? quickPrice : fullPrice;
  if (!(resolvedPrice > 0)) return null;

  const rentNum = Number(params.get("rent"));
  const price = `物件価格 ${resolvedPrice.toLocaleString()}万円`;
  const rent = rentNum > 0 ? `月額賃料 ${rentNum.toLocaleString()}円` : null;
  return [price, rent].filter(Boolean).join(" / ");
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url);

    if (url.pathname.startsWith("/api/")) {
      if (!env.BACKEND_URL) return new Response("BACKEND_URL is not configured", { status: 502 });
      const upstream = new Request(env.BACKEND_URL + url.pathname + url.search, request);
      upstream.headers.delete("host");
      if (env.APP_INTERNAL_API_KEY)
        upstream.headers.set("X-Internal-Key", env.APP_INTERNAL_API_KEY);
      return fetch(upstream, { redirect: "manual" });
    }

    const res = await env.ASSETS.fetch(request);
    const description = url.pathname === "/" ? shareDescription(url.searchParams) : null;
    if (!description) return res;

    return new HTMLRewriter()
      .on("title", {
        element: (el) => {
          el.setInnerContent(`${description} — Yield-Guard シミュレーション結果`);
        },
      })
      .on('meta[name="description"]', {
        element: (el) => {
          el.setAttribute("content", `${description} のシミュレーション結果を確認する`);
          // description は数値由来の固定文言のみなのでエスケープ不要
          el.after(
            `<meta property="og:title" content="${description} — Yield-Guard">` +
              `<meta property="og:description" content="利回り・デッドクロス・出口戦略を分析した結果です">`,
            { html: true }
          );
        },
      })
      .transform(res);
  },
} satisfies ExportedHandler<Env>;
