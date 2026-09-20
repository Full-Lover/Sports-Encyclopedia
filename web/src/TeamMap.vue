<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref } from "vue";
import L from "leaflet";
import "leaflet/dist/leaflet.css";
import { officialSite, teamPath } from "./teamLinks.js";

const props = defineProps({
  places: { type: Array, required: true },
});

const mapElement = ref(null);
const tileUnavailable = ref(false);
const selectedPlace = ref(null);
const selectedTeam = ref(null);
const closeButton = ref(null);
const firstTeamButton = ref(null);
const shareStatus = ref("");
let map;
let selectedMarker;

const officialURL = computed(() => officialSite(selectedTeam.value?.preview?.actions?.officialWebsiteUrl));
const detailsPath = computed(() => teamPath(selectedTeam.value?.preview?.actions?.detailsPath));
const sharePath = computed(() => {
  const path = teamPath(selectedTeam.value?.preview?.actions?.sharePath);
  return path === detailsPath.value ? path : null;
});

function selectTeam(team) {
  selectedTeam.value = team;
  shareStatus.value = "";
  nextTick(() => closeButton.value?.focus());
}

function captureFirstTeamButton(element, index) {
  if (index === 0) firstTeamButton.value = element;
}

function selectPlace(place, marker) {
  selectedPlace.value = place;
  selectedMarker = marker;
  if (place.teams.length === 1) selectTeam(place.teams[0]);
  else {
    selectedTeam.value = null;
    nextTick(() => firstTeamButton.value?.focus());
  }
}

function closePreview() {
  selectedPlace.value = null;
  selectedTeam.value = null;
  shareStatus.value = "";
  nextTick(() => selectedMarker?.getElement()?.focus());
}

async function shareTeam() {
  if (!sharePath.value) return;
  const url = new URL(sharePath.value, window.location.origin).href;
  const title = `${selectedTeam.value.name} | Sports Encyclopedia`;
  try {
    if (navigator.share) {
      await navigator.share({ title, url });
      shareStatus.value = "Shared.";
      return;
    }
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(url);
      shareStatus.value = "Link copied.";
      return;
    }
  } catch (error) {
    if (error?.name === "AbortError") return;
  }
  shareStatus.value = `Copy this link: ${url}`;
}

const tileURL = import.meta.env.VITE_MAP_TILE_URL || "https://tile.openstreetmap.org/{z}/{x}/{y}.png";
const attribution = import.meta.env.VITE_MAP_TILE_ATTRIBUTION ||
  '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap contributors</a>';

onMounted(() => {
  map = L.map(mapElement.value, { scrollWheelZoom: false });
  map.setView([43, -96], 4);
  L.tileLayer(tileURL, { attribution, maxZoom: 19 })
    .on("tileerror", () => { tileUnavailable.value = true; })
    .addTo(map);

  const positions = [];
  for (const place of props.places) {
    const { latitude, longitude } = place.coordinates ?? {};
    if (!Number.isFinite(latitude) || !Number.isFinite(longitude) ||
        Math.abs(latitude) > 90 || Math.abs(longitude) > 180 || !place.teams.length) continue;

    const mark = document.createElement("span");
    mark.textContent = place.teams.length === 1 ? place.teams[0].visual.text : String(place.teams.length);
    const icon = L.divIcon({
      className: "team-map-marker",
      html: mark,
      iconSize: [48, 48],
      iconAnchor: [24, 24],
    });
    const marker = L.marker([latitude, longitude], {
      icon,
      title: place.accessibleName,
      keyboard: true,
    }).addTo(map);
    const tooltip = document.createElement("span");
    tooltip.textContent = place.accessibleName;
    marker.bindTooltip(tooltip);
    marker.on("click", () => selectPlace(place, marker));
    positions.push([latitude, longitude]);
  }

  if (positions.length === 1) map.setView(positions[0], 6);
  if (positions.length > 1) map.fitBounds(positions, { padding: [36, 36], maxZoom: 6 });
});

onUnmounted(() => map?.remove());
</script>

<template>
  <div>
    <div class="team-map-frame">
      <div ref="mapElement" class="team-map" role="region" aria-label="Map of team home venues"></div>
      <section v-if="selectedPlace" class="team-preview" :aria-label="selectedTeam ? 'Selected team' : 'Choose a team at this venue'">
        <button ref="closeButton" type="button" class="preview-close" aria-label="Close team preview" @click="closePreview">×</button>
        <template v-if="selectedTeam">
          <div class="preview-photo" role="img" :aria-label="selectedTeam.preview?.venuePhoto?.alt || 'Venue photo unavailable'">
            <span>Venue photo unavailable</span>
          </div>
          <div class="preview-identity">
            <span class="preview-mark" aria-hidden="true">{{ selectedTeam.visual?.text }}</span>
            <h3>{{ selectedTeam.name }}</h3>
          </div>
          <div class="preview-venue">
            <p>Home venue</p>
            <h4>{{ selectedTeam.venueName }}</h4>
          </div>
          <div class="preview-actions">
            <a v-if="officialURL" :href="officialURL" target="_blank" rel="noopener noreferrer">Official website</a>
            <button v-if="sharePath" type="button" @click="shareTeam">Share</button>
            <a v-if="detailsPath" :href="detailsPath">View details</a>
          </div>
          <p v-if="shareStatus" class="preview-share-status" role="status">{{ shareStatus }}</p>
        </template>
        <div v-else class="preview-chooser">
          <h3>Choose a team at {{ selectedPlace.accessibleName }}</h3>
          <button v-for="(team, index) in selectedPlace.teams" :key="team.teamId"
            :ref="(element) => captureFirstTeamButton(element, index)"
            type="button" @click="selectTeam(team)">{{ team.name }}</button>
        </div>
      </section>
    </div>
    <p v-if="tileUnavailable" class="map-warning" role="status">Map tiles are unavailable. You can still browse the team directory below.</p>
  </div>
</template>
