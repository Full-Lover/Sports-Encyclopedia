import { mount } from "@vue/test-utils";
import { afterEach, describe, expect, it, vi } from "vitest";

import TeamMap from "./TeamMap.vue";

const celtics = {
  teamId: "nba-boston-celtics",
  name: "Boston Celtics",
  venueName: "TD Garden",
  visual: { text: "BOS" },
  preview: {
    venuePhoto: { kind: "PLACEHOLDER", alt: "TD Garden photo unavailable" },
    regularGameCapacity: 19156,
    openedYear: 1995,
    actions: {
      officialWebsiteUrl: "https://www.nba.com/celtics/",
      sharePath: "/teams/boston-celtics",
      detailsPath: "/teams/boston-celtics",
    },
  },
};

const place = {
  venueId: "td-garden",
  accessibleName: "TD Garden, home of Boston Celtics",
  coordinates: { latitude: 42.366303, longitude: -71.062228 },
  teams: [celtics],
};

afterEach(() => vi.unstubAllGlobals());

describe("TeamMap preview", () => {
  it("opens one team's preview with official, share, and detail actions", async () => {
    const wrapper = mount(TeamMap, { props: { places: [place] } });
    await wrapper.get(".leaflet-marker-icon").trigger("click");

    expect(wrapper.get(".team-preview").text()).toContain("Boston Celtics");
    expect(wrapper.get(".team-preview").text()).toContain("TD Garden");
    expect(wrapper.get(".preview-facts").text()).toContain("19,156");
    expect(wrapper.get(".preview-facts").text()).toContain("1995");
    expect(wrapper.get(".preview-photo").text()).toContain("Venue photo unavailable");
    expect(wrapper.get('a[href="https://www.nba.com/celtics/"]').exists()).toBe(true);
    expect(wrapper.get('a[href="/teams/boston-celtics"]').text()).toBe("View details");
    expect(wrapper.find(".team-preview").text()).not.toContain("Directions");

    await wrapper.get('.preview-close').trigger("click");
    expect(wrapper.find(".team-preview").exists()).toBe(false);
    wrapper.unmount();
  });

  it("marks missing or invalid venue facts as unavailable", async () => {
    const incomplete = structuredClone(celtics);
    delete incomplete.preview.regularGameCapacity;
    incomplete.preview.openedYear = 3000;
    const wrapper = mount(TeamMap, { props: { places: [{ ...place, teams: [incomplete] }] } });
    await wrapper.get(".leaflet-marker-icon").trigger("click");

    expect(wrapper.findAll(".preview-facts dd").map((value) => value.text()))
      .toEqual(["Not available", "Not available"]);
    wrapper.unmount();
  });

  it("asks which team to preview at a shared venue", async () => {
    const bruins = { ...celtics, teamId: "nhl-boston-bruins", name: "Boston Bruins" };
    const wrapper = mount(TeamMap, { props: { places: [{ ...place, teams: [celtics, bruins] }] } });
    await wrapper.get(".leaflet-marker-icon").trigger("click");

    expect(wrapper.get(".preview-chooser").text()).toContain("Boston Bruins");
    expect(wrapper.get(".team-preview").attributes("aria-label")).toBe("Choose a team at this venue");
    expect(wrapper.find(".preview-venue").exists()).toBe(false);
    await wrapper.findAll(".preview-chooser button")[1].trigger("click");
    expect(wrapper.get(".preview-identity").text()).toContain("Boston Bruins");
    wrapper.unmount();
  });

  it("omits an unsafe official link and provides a manual share link when browser sharing is unavailable", async () => {
    const unsafe = structuredClone(celtics);
    unsafe.preview.actions.officialWebsiteUrl = "javascript:alert(1)";
    const wrapper = mount(TeamMap, { props: { places: [{ ...place, teams: [unsafe] }] } });
    await wrapper.get(".leaflet-marker-icon").trigger("click");

    expect(wrapper.find('a[href^="javascript:"]').exists()).toBe(false);
    await wrapper.get(".preview-actions button").trigger("click");
    expect(wrapper.get('[role="status"]').text()).toContain("/teams/boston-celtics");
    wrapper.unmount();
  });

  it("does not share a different team's path", async () => {
    const mismatched = structuredClone(celtics);
    mismatched.preview.actions.sharePath = "/teams/another-team";
    const wrapper = mount(TeamMap, { props: { places: [{ ...place, teams: [mismatched] }] } });
    await wrapper.get(".leaflet-marker-icon").trigger("click");

    expect(wrapper.get('.preview-actions a[href="/teams/boston-celtics"]').exists()).toBe(true);
    expect(wrapper.find(".preview-actions button").exists()).toBe(false);
    wrapper.unmount();
  });

  it("omits unsafe detail and share paths", async () => {
    const unsafe = structuredClone(celtics);
    unsafe.preview.actions.detailsPath = "//another-site.example/team";
    const wrapper = mount(TeamMap, { props: { places: [{ ...place, teams: [unsafe] }] } });
    await wrapper.get(".leaflet-marker-icon").trigger("click");

    expect(wrapper.find('.preview-actions a[href^="//"]').exists()).toBe(false);
    expect(wrapper.find(".preview-actions button").exists()).toBe(false);
    wrapper.unmount();
  });
});
