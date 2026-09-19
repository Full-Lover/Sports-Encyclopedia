import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, describe, expect, it, vi } from "vitest";

import App from "./App.vue";

const mapDocument = {
  snapshotId: "preview-0001",
  leagues: [
    { code: "NBA", name: "National Basketball Association" },
    { code: "NFL", name: "National Football League" },
  ],
  places: [{
    teams: [{
      teamId: "nba-boston-celtics",
      name: "Boston Celtics",
      league: "NBA",
      venueName: "TD Garden",
      visual: { text: "BOS" },
      preview: { actions: { officialWebsiteUrl: "https://www.nba.com/celtics/" } },
    }],
  }],
};

afterEach(() => vi.unstubAllGlobals());

describe("App", () => {
  it("loads the page snapshot and shows the available team by league", async () => {
    const fetch = vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => mapDocument });
    vi.stubGlobal("fetch", fetch);
    const wrapper = mount(App, { props: { snapshotId: "preview-0001" } });
    await flushPromises();

    expect(fetch).toHaveBeenCalledWith("/_atlas/snapshots/preview-0001/map");
    expect(wrapper.get("h1").text()).toBe("Explore teams by league");
    expect(wrapper.text()).toContain("Boston Celtics");
    expect(wrapper.text()).toContain("TD Garden");
    expect(wrapper.text()).toContain("Teams from this league are not available in the preview yet.");
    expect(wrapper.get("a").attributes("href")).toBe("https://www.nba.com/celtics/");
  });

  it("offers a retry when the document cannot be loaded", async () => {
    const fetch = vi.fn()
      .mockRejectedValueOnce(new Error("network unavailable"))
      .mockResolvedValueOnce({ ok: true, status: 200, json: async () => mapDocument });
    vi.stubGlobal("fetch", fetch);
    const wrapper = mount(App, { props: { snapshotId: "preview-0001" } });
    await flushPromises();

    expect(wrapper.get('[role="alert"]').text()).toContain("Team data is unavailable");
    await wrapper.get("button").trigger("click");
    await flushPromises();
    expect(wrapper.text()).toContain("Boston Celtics");
    expect(fetch).toHaveBeenCalledTimes(2);
  });

  it("rejects data from a different snapshot", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ ...mapDocument, snapshotId: "different" }),
    }));
    const wrapper = mount(App, { props: { snapshotId: "preview-0001" } });
    await flushPromises();

    expect(wrapper.get('[role="alert"]').text()).toContain("Team data is unavailable");
    expect(wrapper.text()).not.toContain("Boston Celtics");
  });

  it("shows an error for a malformed place instead of rendering a partial directory", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ ...mapDocument, places: [{}] }),
    }));
    const wrapper = mount(App, { props: { snapshotId: "preview-0001" } });
    await flushPromises();

    expect(wrapper.get('[role="alert"]').text()).toContain("Team data is unavailable");
    expect(wrapper.text()).not.toContain("Boston Celtics");
  });

  it("does not render an unsafe official-site link", async () => {
    const unsafe = structuredClone(mapDocument);
    unsafe.places[0].teams[0].preview.actions.officialWebsiteUrl = "javascript:alert(1)";
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => unsafe }));
    const wrapper = mount(App, { props: { snapshotId: "preview-0001" } });
    await flushPromises();

    expect(wrapper.text()).toContain("Boston Celtics");
    expect(wrapper.find("a").exists()).toBe(false);
  });
});
