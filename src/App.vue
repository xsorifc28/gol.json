<script setup>
import { ref, onMounted, computed, onUnmounted } from 'vue';
import { sofascore } from './services/sofascore';
import { Play, ChevronDown, ChevronRight, RefreshCw, Terminal as TerminalIcon } from 'lucide-vue-next';

const events = ref([]);
const loading = ref(false);
const error = ref(null);
const selectedEventId = ref(null);
const selectedEvent = ref(null);
const incidents = ref([]);
const collapsedLeagues = ref(new Set());
const pollingInterval = ref(null);

const leagues = computed(() => {
  const groups = {};
  events.value.forEach(e => {
    const leagueName = `${e.tournament.category.name}: ${e.tournament.name}`;
    if (!groups[leagueName]) groups[leagueName] = [];
    groups[leagueName].push(e);
  });
  return Object.keys(groups).sort().map(name => ({ name, events: groups[name] }));
});

const jsonOutput = computed(() => {
  if (!selectedEvent.value) return null;
  const e = selectedEvent.value;
  let minute = "";
  if (e.statusTime) {
    const now = Math.floor(Date.now() / 1000);
    const elapsed = now - e.statusTime.timestamp;
    const currentMin = Math.floor((e.statusTime.initial + elapsed) / 60);
    const maxRegular = Math.floor(e.statusTime.max / 60);
    if (currentMin >= maxRegular) {
      const added = currentMin - maxRegular;
      const totalAdded = Math.floor((e.statusTime.extra || 0) / 60);
      minute = `${maxRegular}+${added}${totalAdded > 0 ? ` (+${totalAdded})` : ''}`;
    } else {
      minute = `${currentMin + 1}`;
    }
  }
  const reds = [];
  const yellows = [];
  incidents.value.forEach(inc => {
    if (inc.incidentType === 'card') {
      const team = inc.incidentClass === 'home' ? e.homeTeam.name : e.awayTeam.name;
      const cardStr = `${inc.player?.name || 'Unknown'} (${team})`;
      if (inc.cardType === 'Red' || inc.cardType === 'YellowRed') reds.push(cardStr);
      else if (inc.cardType === 'Yellow') yellows.push(cardStr);
    }
  });
  const out = {
    score: `${e.homeTeam.name} ${e.homeScore?.current || 0}-${e.awayScore?.current || 0} ${e.awayTeam.name}`,
    minute: minute || e.status.description
  };
  if (reds.length > 0) out.redCards = reds.join(', ');
  if (yellows.length > 0) out.yellowCards = yellows.join(', ');
  return JSON.stringify(out, null, 2);
});

async function fetchGames() {
  loading.value = true;
  error.value = null;
  try {
    events.value = await sofascore.getLiveEvents();
  } catch (err) {
    error.value = "Failed to fetch games. Check CORS proxy.";
  } finally {
    loading.value = false;
  }
}

async function updateLiveMatch() {
  if (!selectedEventId.value) return;
  try {
    selectedEvent.value = await sofascore.getEventDetails(selectedEventId.value);
    incidents.value = await sofascore.getIncidents(selectedEventId.value);
  } catch (err) {}
}

function selectEvent(id) {
  selectedEventId.value = id;
  updateLiveMatch();
  if (pollingInterval.value) clearInterval(pollingInterval.value);
  pollingInterval.value = setInterval(updateLiveMatch, 10000);
}

function toggleLeague(name) {
  if (collapsedLeagues.value.has(name)) collapsedLeagues.value.delete(name);
  else collapsedLeagues.value.add(name);
}

onMounted(fetchGames);
onUnmounted(() => { if (pollingInterval.value) clearInterval(pollingInterval.value); });
</script>

<template>
  <div class="min-h-screen flex flex-col font-sans p-4 max-w-4xl mx-auto">
    <header class="flex items-center justify-between mb-8">
      <div class="flex items-center gap-2">
        <TerminalIcon class="text-green-500" />
        <h1 class="text-2xl font-bold tracking-tight">gol.json</h1>
      </div>
      <button @click="fetchGames" class="p-2 hover:bg-zinc-800 rounded-full transition-colors" :class="{ 'animate-spin': loading }">
        <RefreshCw size="20" />
      </button>
    </header>
    <div v-if="error" class="bg-red-900/30 border border-red-500 text-red-200 p-4 rounded-lg mb-6 text-sm">
      {{ error }}
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
      <div class="flex flex-col gap-4 overflow-y-auto max-h-[70vh]">
        <div v-for="league in leagues" :key="league.name" class="flex flex-col">
          <button @click="toggleLeague(league.name)" class="flex items-center gap-2 text-zinc-400 font-bold text-[10px] uppercase tracking-widest py-2 hover:text-zinc-200">
            <ChevronDown v-if="!collapsedLeagues.has(league.name)" size="12" />
            <ChevronRight v-else size="12" />
            {{ league.name }}
          </button>
          <div v-if="!collapsedLeagues.has(league.name)" class="flex flex-col gap-1 ml-2 border-l border-zinc-800">
            <button v-for="e in league.events" :key="e.id" @click="selectEvent(e.id)" class="text-left px-3 py-2 rounded transition-colors flex justify-between items-center group" :class="selectedEventId === e.id ? 'bg-zinc-800 text-green-400' : 'hover:bg-zinc-900'">
              <div class="flex flex-col">
                <span class="text-sm">{{ e.homeTeam.name }} vs {{ e.awayTeam.name }}</span>
                <span class="text-[10px] text-zinc-500 uppercase">{{ e.status.description }}</span>
              </div>
              <div class="flex items-center gap-3">
                 <span class="font-mono font-bold">{{ e.homeScore?.current || 0 }} - {{ e.awayScore?.current || 0 }}</span>
                 <Play size="10" class="opacity-0 group-hover:opacity-100 transition-opacity" />
              </div>
            </button>
          </div>
        </div>
      </div>
      <div class="flex flex-col gap-4">
        <h2 class="text-[10px] font-bold text-zinc-500 uppercase tracking-widest">E-Paper Mock (JSON)</h2>
        <div v-if="jsonOutput" class="relative group">
           <pre class="bg-zinc-900 border border-zinc-800 p-6 rounded-xl font-mono text-sm overflow-x-auto text-green-400 shadow-2xl min-h-[200px]"><code>{{ jsonOutput }}</code></pre>
           <div class="absolute top-4 right-4 flex items-center gap-2">
              <span class="flex h-2 w-2 rounded-full bg-green-500 animate-pulse"></span>
              <span class="text-[10px] text-zinc-500 uppercase font-bold">Live</span>
           </div>
        </div>
        <div v-else class="h-64 flex items-center justify-center border-2 border-dashed border-zinc-800 rounded-xl text-zinc-600 italic text-sm">
          Select a game to start monitoring
        </div>
      </div>
    </div>
  </div>
</template>
