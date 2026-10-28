import { describe, expect, it } from "vitest";
import { boxToPercent, cropWindow, pairProblem, reviewProblem, tallyReviews, type Asset, type Finding } from "./types";

const capture = { viewportWidthCss: 320, viewportHeightCss: 200, devicePixelRatio: 2, contentStateKey: "default" };
const asset = (role: Asset["role"], over: Partial<Asset["capture"]> = {}): Asset => ({
  id: role,
  projectId: "p",
  role,
  sha256: "",
  mime: "image/png",
  bytes: 1,
  widthPx: 640,
  heightPx: 400,
  capture: { ...capture, ...over },
  contentUrl: "",
  createdAt: "",
});

describe("boxToPercent", () => {
  it("maps original pixels to display percentages", () => {
    expect(boxToPercent({ x: 160, y: 100, width: 320, height: 200 }, 640, 400)).toEqual({
      left: 25,
      top: 25,
      width: 50,
      height: 50,
    });
  });
});

describe("cropWindow", () => {
  it("pads the box and clamps to the image", () => {
    expect(cropWindow({ x: 100, y: 100, width: 100, height: 40 }, 640, 400)).toEqual({ x: 65, y: 84, width: 170, height: 72 });
    expect(cropWindow({ x: 0, y: 380, width: 50, height: 20 }, 640, 400)).toEqual({ x: 0, y: 364, width: 68, height: 36 });
  });
});

describe("pairProblem", () => {
  it("accepts a matching pair", () => {
    expect(pairProblem(asset("reference"), asset("actual"))).toBeNull();
  });
  it("rejects capture mismatch and swapped roles", () => {
    expect(pairProblem(asset("reference"), asset("actual", { devicePixelRatio: 1 }))).toMatch(/must match/);
    expect(pairProblem(asset("actual"), asset("reference"))).toMatch(/swapped/);
    expect(pairProblem(undefined, asset("actual"))).toMatch(/Choose/);
  });
});

describe("reviews", () => {
  it("requires a reason to reject", () => {
    expect(reviewProblem("rejected", "  ")).not.toBeNull();
    expect(reviewProblem("accepted", "")).toBeNull();
  });
  it("tallies current decisions", () => {
    const f = (decision: "accepted" | "rejected" | null) =>
      ({ currentReview: decision ? { decision } : null }) as unknown as Finding;
    expect(tallyReviews([f("accepted"), f("rejected"), f(null), f("accepted")])).toEqual({
      total: 4,
      accepted: 2,
      rejected: 1,
      unreviewed: 1,
    });
  });
});
