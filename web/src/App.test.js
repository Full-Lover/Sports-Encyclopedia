import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, describe, expect, it, vi } from "vitest";

import App from "./App.vue";

vi.mock("./TeamMap.vue", () => ({
  default: {
    props: ["places"],
    template: '<div data-test-map>{{ places.flatMap((place) => place.teams.map((team) => team.name)).join(", ") }}</div>',
  },
}));

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
      officialGroup: "Eastern Conference",
      division: "Atlantic Division",
      venueName: "TD Garden",
      visual: { text: "BOS" },
      preview: { actions: {
        officialWebsiteUrl: "https://www.nba.com/celtics/",
        detailsPath: "/teams/boston-celtics",
      } },
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
    expect(wrapper.get("h1").text()).toBe("Explore teams by place");
    expect(wrapper.get("[data-test-map]").exists()).toBe(true);
    expect(wrapper.find(".directory").exists()).toBe(false);
    await wrapper.findAll(".view-switch button")[1].trigger("click");
    expect(wrapper.find("[data-test-map]").exists()).toBe(false);
    expect(wrapper.text()).toContain("Boston Celtics");
    expect(wrapper.text()).toContain("Eastern Conference");
    expect(wrapper.text()).toContain("Atlantic Division");
    expect(wrapper.text()).toContain("TD Garden");
    expect(wrapper.text()).toContain("Teams from this league are not available in the preview yet.");
    expect(wrapper.get('a[aria-label="Boston Celtics official website (opens in a new tab)"]').attributes("href"))
      .toBe("https://www.nba.com/celtics/");
    expect(wrapper.get(".details-link").attributes("href")).toBe("/teams/boston-celtics");
    expect(wrapper.get(".details-link").attributes("aria-label")).toBe("View Boston Celtics details");
  });

  it("organizes teams by official group and division", async () => {
    const grouped = structuredClone(mapDocument);
    grouped.places.push({ teams: [
      { teamId: "nba-chicago-bulls", name: "Chicago Bulls", league: "NBA",
        officialGroup: "Eastern Conference", division: "Central Division",
        venueName: "United Center", visual: { text: "CHI" } },
      { teamId: "nba-los-angeles-lakers", name: "Los Angeles Lakers", league: "NBA",
        officialGroup: "Western Conference", division: "Pacific Division",
        venueName: "Crypto.com Arena", visual: { text: "LAL" } },
    ] });
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => grouped }));
    const wrapper = mount(App, { props: { snapshotId: "preview-0001" } });
    await flushPromises();
    await wrapper.findAll(".view-switch button")[1].trigger("click");

    const nba = wrapper.findAll(".league-section")[0];
    expect(nba.findAll(".official-group h5").map((heading) => heading.text()))
      .toEqual(["Eastern Conference", "Western Conference"]);
    expect(nba.findAll(".division h6").map((heading) => heading.text()))
      .toEqual(["Atlantic Division", "Central Division", "Pacific Division"]);
    expect(nba.findAll(".division .team-name").map((name) => name.text()))
      .toEqual(["Boston Celtics", "Chicago Bulls", "Los Angeles Lakers"]);
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

  it("filters the map and directory together and restores all leagues", async () => {
    const withNFL = structuredClone(mapDocument);
    withNFL.places.push({
      teams: [{
        teamId: "nfl-new-york-giants", name: "New York Giants", league: "NFL",
        venueName: "MetLife Stadium", visual: { text: "NYG" },
        preview: { actions: { officialWebsiteUrl: "https://www.giants.com/" } },
      }],
    });
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => withNFL }));
    const wrapper = mount(App, { props: { snapshotId: "preview-0001" } });
    await flushPromises();

    expect(wrapper.get("[data-test-map]").text()).toContain("Boston Celtics");
    expect(wrapper.get("[data-test-map]").text()).toContain("New York Giants");
    await wrapper.findAll(".league-filter")[0].trigger("click");
    expect(wrapper.get("[data-test-map]").text()).not.toContain("Boston Celtics");
    expect(wrapper.get("[data-test-map]").text()).toContain("New York Giants");
    await wrapper.findAll(".view-switch button")[1].trigger("click");
    expect(wrapper.find("[data-test-map]").exists()).toBe(false);
    expect(wrapper.text()).not.toContain("TD Garden");
    expect(wrapper.text()).toContain("MetLife Stadium");
    expect(wrapper.text()).toContain("Alignment unavailable");
    expect(wrapper.text()).toContain("Division unavailable");

    expect(wrapper.findAll(".league-filter").map((button) => button.attributes("aria-pressed")))
      .toEqual(["false", "true"]);
    await wrapper.findAll(".league-filter")[1].trigger("click");
    expect(wrapper.text()).toContain("Select a league to see teams.");
    await wrapper.get(".show-all").trigger("click");
    await wrapper.findAll(".view-switch button")[0].trigger("click");
    expect(wrapper.get("[data-test-map]").text()).toContain("Boston Celtics");
    expect(wrapper.get("[data-test-map]").text()).toContain("New York Giants");
    expect(fetch).toHaveBeenCalledTimes(1);
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
    await wrapper.findAll(".view-switch button")[1].trigger("click");

    expect(wrapper.text()).toContain("Boston Celtics");
    expect(wrapper.find('a[aria-label="Boston Celtics official website (opens in a new tab)"]').exists()).toBe(false);
  });

  it("does not render an unsafe team-details link", async () => {
    const unsafe = structuredClone(mapDocument);
    unsafe.places[0].teams[0].preview.actions.detailsPath = "//another-site.example/team";
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => unsafe }));
    const wrapper = mount(App, { props: { snapshotId: "preview-0001" } });
    await flushPromises();
    await wrapper.findAll(".view-switch button")[1].trigger("click");

    expect(wrapper.text()).toContain("Boston Celtics");
    expect(wrapper.find(".details-link").exists()).toBe(false);
  });
});
