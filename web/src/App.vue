<script setup>
import { computed, onMounted, ref } from "vue";
import TeamMap from "./TeamMap.vue";

const props = defineProps({
  snapshotId: { type: String, required: true },
});

const document = ref(null);
const state = ref("loading");

function officialSite(url) {
  try {
    const parsed = new URL(url);
    return parsed.protocol === "https:" ? parsed.href : null;
  } catch {
    return null;
  }
}

const leagues = computed(() => {
  if (!document.value) return [];
  return document.value.leagues.map((league) => ({
    ...league,
    teams: document.value.places.flatMap((place) =>
      place.teams.filter((team) => team.league === league.code).map((team) => ({
        ...team,
        officialSite: officialSite(team.preview?.actions?.officialWebsiteUrl),
      })),
    ),
  }));
});

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

    <section v-if="state === 'ready'" class="map-section" aria-labelledby="map-title">
      <div class="map-heading">
        <h2 id="map-title">Team map</h2>
        <a href="#directory-title">Browse teams as a list</a>
      </div>
      <TeamMap :places="document.places" />
    </section>

    <section class="directory" aria-labelledby="directory-title">
      <div class="directory-heading">
        <h2 id="directory-title">Team directory</h2>
        <p>Browse the teams currently included in this preview.</p>
      </div>

      <p v-if="state === 'loading'" class="feedback" role="status">Loading teams…</p>
      <div v-else-if="state === 'error'" class="feedback" role="alert">
        <p>Team data is unavailable. Try again to reload the directory.</p>
        <button type="button" @click="loadMap">Try again</button>
      </div>
      <div v-else class="league-list">
        <section v-for="league in leagues" :key="league.code" class="league-section" :aria-labelledby="`league-${league.code}`">
          <div class="league-heading">
            <h3 :id="`league-${league.code}`">{{ league.code }}</h3>
            <p>{{ league.name }}</p>
            <span class="team-count">{{ league.teams.length }} {{ league.teams.length === 1 ? 'team' : 'teams' }}</span>
          </div>
          <ul v-if="league.teams.length" class="team-list">
            <li v-for="team in league.teams" :key="team.teamId" class="team-row">
              <span class="team-mark" aria-hidden="true">{{ team.visual.text }}</span>
              <div>
                <h4>{{ team.name }}</h4>
                <p>{{ team.venueName }}</p>
              </div>
              <a v-if="team.officialSite" :href="team.officialSite" target="_blank" rel="noopener noreferrer" :aria-label="`${team.name} official website (opens in a new tab)`">Official site</a>
            </li>
          </ul>
          <p v-else class="empty-league">Teams from this league are not available in the preview yet.</p>
        </section>
      </div>
    </section>
  </div>
</template>
