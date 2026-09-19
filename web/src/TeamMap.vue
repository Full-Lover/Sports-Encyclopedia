<script setup>
import { onMounted, onUnmounted, ref } from "vue";
import L from "leaflet";
import "leaflet/dist/leaflet.css";

const props = defineProps({
  places: { type: Array, required: true },
});

const mapElement = ref(null);
const tileUnavailable = ref(false);
let map;

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
    marker.on("click", () => marker.openTooltip());
    positions.push([latitude, longitude]);
  }

  if (positions.length === 1) map.setView(positions[0], 6);
  if (positions.length > 1) map.fitBounds(positions, { padding: [36, 36], maxZoom: 6 });
});

onUnmounted(() => map?.remove());
</script>

<template>
  <div>
    <div ref="mapElement" class="team-map" role="region" aria-label="Map of team home venues"></div>
    <p v-if="tileUnavailable" class="map-warning" role="status">Map tiles are unavailable. You can still browse the team directory below.</p>
  </div>
</template>
