<script setup>
import { computed, onMounted, ref } from "vue";
import TeamMap from "./TeamMap.vue";
import { officialSite, teamPath } from "./teamLinks.js";

const props = defineProps({
  snapshotId: { type: String, required: true },
});

const document = ref(null);
const state = ref("loading");
const selectedLeagues = ref([]);
const viewMode = ref("map");

const leagues = computed(() => {
  if (!document.value) return [];
  return document.value.leagues.map((league) => {
    const teams = document.value.places.flatMap((place) =>
      place.teams.filter((team) => team.league === league.code).map((team) => ({
        ...team,
        officialSite: officialSite(team.preview?.actions?.officialWebsiteUrl),
        detailsPath: teamPath(team.preview?.actions?.detailsPath),
      })),
    );
    const groups = [];
    for (const team of teams) {
      const groupName = typeof team.officialGroup === "string" && team.officialGroup.trim()
        ? team.officialGroup : "Alignment unavailable";
      const divisionName = typeof team.division === "string" && team.division.trim()
        ? team.division : "Division unavailable";
      let group = groups.find((entry) => entry.name === groupName);
      if (!group) {
        group = { name: groupName, divisions: [] };
        groups.push(group);
      }
      let division = group.divisions.find((entry) => entry.name === divisionName);
      if (!division) {
        division = { name: divisionName, teams: [] };
        group.divisions.push(division);
      }
      division.teams.push(team);
    }
    return { ...league, teams, groups };
  });
});

const visibleLeagues = computed(() => leagues.value.filter((league) => selectedLeagues.value.includes(league.code)));
const visiblePlaces = computed(() => (document.value?.places ?? []).flatMap((place) => {
  const teams = place.teams.filter((team) => selectedLeagues.value.includes(team.league));
  if (!teams.length) return [];
  const accessibleName = teams.length === place.teams.length ? place.accessibleName
    : `${teams[0].venueName}, home of ${teams.map((team) => team.name).join(" and ")}`;
  return [{ ...place, accessibleName, teams }];
}));

function toggleLeague(code) {
  selectedLeagues.value = selectedLeagues.value.includes(code)
    ? selectedLeagues.value.filter((selected) => selected !== code)
    : [...selectedLeagues.value, code];
}

function showAllLeagues() {
  selectedLeagues.value = document.value.leagues.map((league) => league.code);
}

async function loadMap() {
  state.value = "loading";
  try {
    const response = await fetch(`/_atlas/snapshots/${encodeURIComponent(props.snapshotId)}/map`);
    if (response.status === 409 || response.status === 410) {
      window.location.reload();
      return;
    }
    if (!response.ok) throw new Error("Map document unavailable");
    const next = await response.json();
    if (next.snapshotId !== props.snapshotId ||
        !Array.isArray(next.leagues) ||
        !Array.isArray(next.places) ||
        next.places.some((place) => !Array.isArray(place.teams))) {
      throw new Error("Map document does not match this page");
    }
    document.value = next;
    showAllLeagues();
    state.value = "ready";
  } catch {
    state.value = "error";
  }
}

onMounted(loadMap);
</script>

<template>
  <div class="app-shell">
    <header class="page-header">
      <p class="project-name">Sports Encyclopedia</p>
      <h1>Explore teams by place</h1>
      <p class="page-intro">A growing guide to the geography of North American professional sports.</p>
      <p class="preview-note">Preview data — league coverage is limited.</p>
    </header>

    <section class="explorer" aria-labelledby="explorer-title">
      <div class="explorer-heading">
        <h2 id="explorer-title">Explore teams</h2>
        <div v-if="state === 'ready'" class="view-switch" role="group" aria-label="Choose view">
          <button type="button" :aria-pressed="viewMode === 'map'" @click="viewMode = 'map'">Map</button>
          <button type="button" :aria-pressed="viewMode === 'list'" @click="viewMode = 'list'">List</button>
        </div>
      </div>
      <div v-if="state === 'ready'" class="league-filters" role="group" aria-label="Filter by league">
        <button v-for="league in document.leagues" :key="league.code" type="button"
          class="league-filter" :aria-pressed="selectedLeagues.includes(league.code)"
          @click="toggleLeague(league.code)">{{ league.code }}</button>
        <button type="button" class="show-all" @click="showAllLeagues">Show all</button>
      </div>
      <p v-if="state === 'loading'" class="feedback" role="status">Loading teams…</p>
      <div v-else-if="state === 'error'" class="feedback" role="alert">
        <p>Team data is unavailable. Try again to reload it.</p>
        <button type="button" @click="loadMap">Try again</button>
      </div>
      <div v-else-if="viewMode === 'map'" class="map-view">
        <p v-if="selectedLeagues.length === 0" class="empty-selection" role="status">Select a league to see teams.</p>
        <TeamMap :key="selectedLeagues.join(',')" :places="visiblePlaces" />
      </div>
      <div v-else class="directory" aria-labelledby="directory-title">
        <div class="directory-heading">
          <h3 id="directory-title">Team directory</h3>
          <p>Browse the teams currently included in this preview.</p>
        </div>
        <div class="league-list">
          <p v-if="visibleLeagues.length === 0" class="empty-selection">Select a league to see teams.</p>
          <section v-for="league in visibleLeagues" :key="league.code" class="league-section" :aria-labelledby="`league-${league.code}`">
            <div class="league-heading">
              <h4 :id="`league-${league.code}`">{{ league.code }}</h4>
              <p>{{ league.name }}</p>
              <span class="team-count">{{ league.teams.length }} {{ league.teams.length === 1 ? 'team' : 'teams' }}</span>
            </div>
            <template v-if="league.teams.length">
              <div v-for="group in league.groups" :key="group.name" class="official-group">
                <h5>{{ group.name }}</h5>
                <div v-for="division in group.divisions" :key="division.name" class="division">
                  <h6>{{ division.name }}</h6>
                  <ul class="team-list">
                    <li v-for="team in division.teams" :key="team.teamId" class="team-row">
                      <span class="team-mark" aria-hidden="true">{{ team.visual.text }}</span>
                      <div>
                        <strong class="team-name">{{ team.name }}</strong>
                        <p>{{ team.venueName }}</p>
                      </div>
                      <a v-if="team.detailsPath" class="details-link" :href="team.detailsPath" :aria-label="`View ${team.name} details`">View details</a>
                      <a v-if="team.officialSite" :href="team.officialSite" target="_blank" rel="noopener noreferrer" :aria-label="`${team.name} official website (opens in a new tab)`">Official site</a>
                    </li>
                  </ul>
                </div>
              </div>
            </template>
            <p v-else class="empty-league">Teams from this league are not available in the preview yet.</p>
          </section>
        </div>
      </div>
    </section>
  </div>
</template>
