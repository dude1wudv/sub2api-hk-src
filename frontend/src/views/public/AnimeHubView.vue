<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import Icon from '@/components/icons/Icon.vue'

type MediaSeason = 'WINTER' | 'SPRING' | 'SUMMER' | 'FALL'
type LoadingState = 'idle' | 'loading' | 'ready' | 'error'

interface AnimeTitle {
  userPreferred?: string | null
  romaji?: string | null
  english?: string | null
  native?: string | null
}

interface AnimeMedia {
  id: number
  idMal?: number | null
  title: AnimeTitle
  coverImage?: {
    large?: string | null
    color?: string | null
  } | null
  bannerImage?: string | null
  siteUrl?: string | null
  format?: string | null
  episodes?: number | null
  status?: string | null
  averageScore?: number | null
  popularity?: number | null
  genres?: string[] | null
  isAdult?: boolean | null
  externalLinks?: AnimeExternalLink[] | null
}

interface AiringSchedule {
  id: number
  airingAt: number
  episode: number
  media: AnimeMedia
}

interface AnimeCard {
  id: string
  mediaId: number
  title: string
  nativeTitle: string
  episodeLabel: string
  timeLabel: string
  dayIndex: number
  timestamp: number
  cover: string
  banner: string
  accent: string
  score: string
  popularity: string
  format: string
  genres: string[]
  siteUrl: string
  officialLinks: WatchLink[]
}

interface WatchLink {
  label: string
  href: string
  tone: string
}

interface AnimeExternalLink {
  site?: string | null
  url?: string | null
  type?: string | null
  language?: string | null
  isDisabled?: boolean | null
}

interface OfficialStreamingSource {
  label: string
  siteNames: string[]
  hosts: string[]
  tone: string
}

const anilistEndpoint = 'https://graphql.anilist.co'
const fallbackCover = 'https://s4.anilist.co/file/anilistcdn/media/anime/cover/large/bx16498-buvcRTBx4NSm.jpg'
const fallbackBanner = 'https://s4.anilist.co/file/anilistcdn/media/anime/banner/16498.jpg'
const sampleVideoSrc = 'https://media.w3.org/2010/05/sintel/trailer.mp4'
const sampleVideoPoster = 'https://media.w3.org/2010/05/sintel/poster.png'

const officialStreamingSources: OfficialStreamingSource[] = [
  {
    label: 'Crunchyroll',
    siteNames: ['crunchyroll'],
    hosts: ['crunchyroll.com'],
    tone: 'bg-orange-50 text-orange-700 ring-orange-200',
  },
  {
    label: 'Bilibili 番剧',
    siteNames: ['bilibili', 'bilibili tv'],
    hosts: ['bilibili.com', 'bilibili.tv'],
    tone: 'bg-pink-50 text-pink-700 ring-pink-200',
  },
  {
    label: 'Netflix',
    siteNames: ['netflix'],
    hosts: ['netflix.com'],
    tone: 'bg-zinc-100 text-zinc-800 ring-zinc-200',
  },
  {
    label: 'Prime Video',
    siteNames: ['prime video', 'amazon prime video'],
    hosts: ['primevideo.com', 'amazon.com'],
    tone: 'bg-cyan-50 text-cyan-800 ring-cyan-200',
  },
  {
    label: 'Hulu',
    siteNames: ['hulu'],
    hosts: ['hulu.com'],
    tone: 'bg-lime-50 text-lime-800 ring-lime-200',
  },
  {
    label: 'HIDIVE',
    siteNames: ['hidive'],
    hosts: ['hidive.com'],
    tone: 'bg-indigo-50 text-indigo-700 ring-indigo-200',
  },
  {
    label: 'Disney+',
    siteNames: ['disney plus', 'disney+'],
    hosts: ['disneyplus.com'],
    tone: 'bg-blue-50 text-blue-700 ring-blue-200',
  },
  {
    label: 'iQIYI',
    siteNames: ['iq', 'iqiyi', 'iQIYI'],
    hosts: ['iq.com', 'iqiyi.com'],
    tone: 'bg-emerald-50 text-emerald-700 ring-emerald-200',
  },
  {
    label: 'YouTube 官方',
    siteNames: ['youtube'],
    hosts: ['youtube.com', 'youtu.be'],
    tone: 'bg-red-50 text-red-700 ring-red-200',
  },
]

const dayLabels = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'] as const
const dayNames = ['周一', '周二', '周三', '周四', '周五', '周六', '周日'] as const

const loadingState = ref<LoadingState>('idle')
const loadError = ref('')
const schedules = ref<AnimeCard[]>([])
const trending = ref<AnimeCard[]>([])
const selectedDay = ref(getMondayBasedDayIndex(new Date()))
const featuredId = ref('')
const lastUpdated = ref('')

const weekRange = computed(() => getWeekRange(new Date()))
const selectedDayName = computed(() => dayNames[selectedDay.value])
const statusText = computed(() => {
  if (loadingState.value === 'loading') return '同步中'
  if (loadingState.value === 'error') return '使用备用数据'
  return '实时公开数据'
})

const groupedSchedule = computed(() => {
  return dayLabels.map((_, index) => schedules.value
    .filter((item) => item.dayIndex === index)
    .sort((a, b) => a.timestamp - b.timestamp))
})

const selectedSchedule = computed(() => groupedSchedule.value[selectedDay.value] ?? [])
const featuredAnime = computed(() => {
  const selected = selectedSchedule.value.find((item) => item.id === featuredId.value)
  return selected ?? selectedSchedule.value[0] ?? trending.value[0] ?? schedules.value[0]
})

const watchLinks = computed<WatchLink[]>(() => {
  const title = featuredAnime.value?.title || 'anime'
  const encoded = encodeURIComponent(title)
  const fallbackSearchLinks: WatchLink[] = [
    { label: 'Crunchyroll 搜索', href: `https://www.crunchyroll.com/search?q=${encoded}`, tone: 'bg-orange-50 text-orange-700 ring-orange-200' },
    { label: 'Bilibili 番剧搜索', href: `https://search.bilibili.com/bangumi?keyword=${encoded}`, tone: 'bg-pink-50 text-pink-700 ring-pink-200' },
    { label: 'YouTube 官方搜索', href: `https://www.youtube.com/results?search_query=${encoded}%20official%20anime`, tone: 'bg-red-50 text-red-700 ring-red-200' },
    { label: 'Netflix 搜索', href: `https://www.netflix.com/search?q=${encoded}`, tone: 'bg-zinc-100 text-zinc-800 ring-zinc-200' },
    { label: 'Prime Video 搜索', href: `https://www.primevideo.com/search/ref=atv_nb_sr?phrase=${encoded}`, tone: 'bg-cyan-50 text-cyan-800 ring-cyan-200' },
  ]
  const links: WatchLink[] = [
    { label: 'AniList', href: featuredAnime.value?.siteUrl || 'https://anilist.co/search/anime', tone: 'bg-sky-50 text-sky-700 ring-sky-200' },
    ...(featuredAnime.value?.officialLinks ?? []),
    ...fallbackSearchLinks,
  ]
  return dedupeWatchLinks(links).slice(0, 8)
})

function getMondayBasedDayIndex(date: Date) {
  return (date.getDay() + 6) % 7
}

function getWeekRange(date: Date) {
  const start = new Date(date)
  start.setHours(0, 0, 0, 0)
  start.setDate(start.getDate() - getMondayBasedDayIndex(start))
  const end = new Date(start)
  end.setDate(start.getDate() + 7)
  return {
    start,
    end,
    label: `${formatMonthDay(start)} - ${formatMonthDay(new Date(end.getTime() - 1))}`,
  }
}

function formatMonthDay(date: Date) {
  return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}

function formatTime(timestamp: number) {
  return new Date(timestamp * 1000).toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
}

function currentSeason(date: Date): { season: MediaSeason; year: number } {
  const month = date.getMonth()
  if (month <= 2) return { season: 'WINTER', year: date.getFullYear() }
  if (month <= 5) return { season: 'SPRING', year: date.getFullYear() }
  if (month <= 8) return { season: 'SUMMER', year: date.getFullYear() }
  return { season: 'FALL', year: date.getFullYear() }
}

function normalizeTitle(media: AnimeMedia) {
  return media.title.userPreferred || media.title.english || media.title.romaji || media.title.native || 'Untitled'
}

function normalizeNativeTitle(media: AnimeMedia) {
  return media.title.native || media.title.romaji || media.title.english || ''
}

function mapSchedule(item: AiringSchedule): AnimeCard {
  const media = item.media
  const date = new Date(item.airingAt * 1000)
  return {
    id: `airing-${item.id}`,
    mediaId: media.id,
    title: normalizeTitle(media),
    nativeTitle: normalizeNativeTitle(media),
    episodeLabel: `EP ${item.episode}`,
    timeLabel: formatTime(item.airingAt),
    dayIndex: getMondayBasedDayIndex(date),
    timestamp: item.airingAt,
    cover: media.coverImage?.large || fallbackCover,
    banner: media.bannerImage || media.coverImage?.large || fallbackBanner,
    accent: media.coverImage?.color || '#0f766e',
    score: media.averageScore ? `${media.averageScore}%` : 'N/A',
    popularity: media.popularity ? Intl.NumberFormat('en', { notation: 'compact' }).format(media.popularity) : 'N/A',
    format: media.format?.replace(/_/g, ' ') || 'TV',
    genres: cleanGenres(media.genres),
    siteUrl: media.siteUrl || `https://anilist.co/anime/${media.id}`,
    officialLinks: buildOfficialStreamingLinks(media.externalLinks),
  }
}

function mapMedia(media: AnimeMedia, index: number): AnimeCard {
  return {
    id: `media-${media.id}`,
    mediaId: media.id,
    title: normalizeTitle(media),
    nativeTitle: normalizeNativeTitle(media),
    episodeLabel: media.episodes ? `${media.episodes} episodes` : media.status?.replace(/_/g, ' ') || 'Season title',
    timeLabel: `#${index + 1}`,
    dayIndex: index % 7,
    timestamp: 0,
    cover: media.coverImage?.large || fallbackCover,
    banner: media.bannerImage || media.coverImage?.large || fallbackBanner,
    accent: media.coverImage?.color || '#0f766e',
    score: media.averageScore ? `${media.averageScore}%` : 'N/A',
    popularity: media.popularity ? Intl.NumberFormat('en', { notation: 'compact' }).format(media.popularity) : 'N/A',
    format: media.format?.replace(/_/g, ' ') || 'TV',
    genres: cleanGenres(media.genres),
    siteUrl: media.siteUrl || `https://anilist.co/anime/${media.id}`,
    officialLinks: buildOfficialStreamingLinks(media.externalLinks),
  }
}

function cleanGenres(genres?: string[] | null) {
  return (genres || [])
    .filter((genre) => !['hentai', 'ecchi'].includes(genre.toLowerCase()))
    .slice(0, 3)
}

function buildOfficialStreamingLinks(externalLinks?: AnimeExternalLink[] | null): WatchLink[] {
  const links = (externalLinks || [])
    .filter((link) => link.type === 'STREAMING' && !link.isDisabled)
    .map((link) => {
      const source = findOfficialStreamingSource(link)
      if (!source) return null
      const href = normalizeStreamingUrl(link.url, source.hosts)
      if (!href) return null
      return {
        label: source.label,
        href,
        tone: source.tone,
      }
    })
    .filter((link): link is WatchLink => Boolean(link))

  return dedupeWatchLinks(links).slice(0, 5)
}

function findOfficialStreamingSource(link: AnimeExternalLink): OfficialStreamingSource | null {
  const site = (link.site || '').toLowerCase()
  const host = getUrlHost(link.url)
  return officialStreamingSources.find((source) => {
    const matchesSite = source.siteNames.some((siteName) => site.includes(siteName.toLowerCase()))
    const matchesHost = host ? source.hosts.some((allowedHost) => isHostAllowed(host, allowedHost)) : false
    return matchesSite || matchesHost
  }) || null
}

function normalizeStreamingUrl(value: string | null | undefined, allowedHosts: string[]) {
  if (!value) return ''
  try {
    const url = new URL(value)
    if (!['http:', 'https:'].includes(url.protocol)) return ''
    if (!allowedHosts.some((host) => isHostAllowed(url.hostname, host))) return ''
    url.protocol = 'https:'
    return url.toString()
  } catch {
    return ''
  }
}

function getUrlHost(value: string | null | undefined) {
  if (!value) return ''
  try {
    return new URL(value).hostname
  } catch {
    return ''
  }
}

function isHostAllowed(hostname: string, allowedHost: string) {
  const host = hostname.toLowerCase().replace(/^www\./, '')
  const allowed = allowedHost.toLowerCase().replace(/^www\./, '')
  return host === allowed || host.endsWith(`.${allowed}`)
}

function dedupeWatchLinks(links: WatchLink[]) {
  const seen = new Set<string>()
  return links.filter((link) => {
    const key = `${link.label}:${link.href}`
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })
}

async function fetchAnimeData() {
  loadingState.value = 'loading'
  loadError.value = ''
  const range = weekRange.value
  const { season, year } = currentSeason(new Date())
  const query = `
    query AnimeWeekly($weekStart: Int, $weekEnd: Int, $season: MediaSeason, $seasonYear: Int) {
      schedule: Page(page: 1, perPage: 50) {
        airingSchedules(airingAt_greater: $weekStart, airingAt_lesser: $weekEnd, sort: TIME) {
          id
          airingAt
          episode
          media {
            id
            idMal
            title { userPreferred romaji english native }
            coverImage { large color }
            bannerImage
            siteUrl
            format
            episodes
            status
            averageScore
            popularity
            genres
            isAdult
            externalLinks {
              site
              url
              type
              language
              isDisabled
            }
          }
        }
      }
      trending: Page(page: 1, perPage: 12) {
        media(type: ANIME, sort: TRENDING_DESC, season: $season, seasonYear: $seasonYear, isAdult: false) {
          id
          idMal
          title { userPreferred romaji english native }
          coverImage { large color }
          bannerImage
          siteUrl
          format
          episodes
          status
          averageScore
          popularity
          genres
          isAdult
          externalLinks {
            site
            url
            type
            language
            isDisabled
          }
        }
      }
    }
  `

  try {
    const response = await fetch(anilistEndpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({
        query,
        variables: {
          weekStart: Math.floor(range.start.getTime() / 1000),
          weekEnd: Math.floor(range.end.getTime() / 1000),
          season,
          seasonYear: year,
        },
      }),
    })
    const payload = await response.json()
    if (!response.ok || payload.errors?.length) {
      throw new Error(payload.errors?.[0]?.message || `AniList request failed: ${response.status}`)
    }

    const nextSchedule = (payload.data?.schedule?.airingSchedules || [])
      .filter((item: AiringSchedule) => !item.media.isAdult)
      .map(mapSchedule)
    const nextTrending = (payload.data?.trending?.media || [])
      .filter((media: AnimeMedia) => !media.isAdult)
      .map(mapMedia)
    schedules.value = nextSchedule.length ? nextSchedule : fallbackSchedule
    trending.value = nextTrending.length ? nextTrending : fallbackTrending
    featuredId.value = selectedSchedule.value[0]?.id || schedules.value[0]?.id || ''
    lastUpdated.value = new Date().toLocaleString('zh-CN', { hour12: false })
    loadingState.value = 'ready'
  } catch (error) {
    schedules.value = fallbackSchedule
    trending.value = fallbackTrending
    featuredId.value = fallbackSchedule[0]?.id || ''
    lastUpdated.value = new Date().toLocaleString('zh-CN', { hour12: false })
    loadError.value = error instanceof Error ? error.message : 'AniList request failed'
    loadingState.value = 'error'
  }
}

function selectDay(index: number) {
  selectedDay.value = index
  featuredId.value = selectedSchedule.value[0]?.id || featuredId.value
}

function selectAnime(item: AnimeCard) {
  featuredId.value = item.id
}

const fallbackSchedule: AnimeCard[] = [
  {
    id: 'fallback-1',
    mediaId: 16498,
    title: 'Attack on Titan',
    nativeTitle: '進撃の巨人',
    episodeLabel: 'Official catalog',
    timeLabel: '20:00',
    dayIndex: 0,
    timestamp: 0,
    cover: fallbackCover,
    banner: fallbackBanner,
    accent: '#7c2d12',
    score: '84%',
    popularity: '1M',
    format: 'TV',
    genres: ['Action', 'Drama', 'Fantasy'],
    siteUrl: 'https://anilist.co/anime/16498',
    officialLinks: [],
  },
  {
    id: 'fallback-2',
    mediaId: 1535,
    title: 'DEATH NOTE',
    nativeTitle: 'DEATH NOTE',
    episodeLabel: 'Official catalog',
    timeLabel: '21:30',
    dayIndex: 2,
    timestamp: 0,
    cover: 'https://s4.anilist.co/file/anilistcdn/media/anime/cover/large/bx1535-4r88a1tsBEIz.jpg',
    banner: 'https://s4.anilist.co/file/anilistcdn/media/anime/banner/1535.jpg',
    accent: '#1f2937',
    score: '84%',
    popularity: '1M',
    format: 'TV',
    genres: ['Mystery', 'Psychological', 'Thriller'],
    siteUrl: 'https://anilist.co/anime/1535',
    officialLinks: [],
  },
  {
    id: 'fallback-3',
    mediaId: 21,
    title: 'ONE PIECE',
    nativeTitle: 'ONE PIECE',
    episodeLabel: 'Official catalog',
    timeLabel: '09:30',
    dayIndex: 6,
    timestamp: 0,
    cover: 'https://s4.anilist.co/file/anilistcdn/media/anime/cover/large/bx21-YCDoj1EkAxFn.jpg',
    banner: 'https://s4.anilist.co/file/anilistcdn/media/anime/banner/21.jpg',
    accent: '#2563eb',
    score: '88%',
    popularity: '1M',
    format: 'TV',
    genres: ['Action', 'Adventure', 'Comedy'],
    siteUrl: 'https://anilist.co/anime/21',
    officialLinks: [],
  },
]

const fallbackTrending = fallbackSchedule.map((item, index) => ({
  ...item,
  id: `fallback-trending-${index}`,
  timeLabel: `#${index + 1}`,
}))

onMounted(fetchAnimeData)
</script>

<template>
  <main class="min-h-screen bg-[#f7f8f5] text-zinc-950">
    <section class="border-b border-zinc-200 bg-white">
      <div class="mx-auto flex max-w-7xl flex-col gap-4 px-4 py-5 sm:px-6 lg:px-8">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div class="min-w-0">
            <RouterLink to="/home" class="mb-4 inline-flex items-center gap-2 text-sm font-semibold text-zinc-600 transition hover:text-zinc-950">
              <Icon name="home" size="sm" />
              Sub2API
            </RouterLink>
            <h1 class="text-3xl font-semibold tracking-normal text-zinc-950 sm:text-4xl">Anime Weekly</h1>
            <p class="mt-2 max-w-2xl text-sm leading-6 text-zinc-600">
              新番时间表、热门条目和官方观看入口。数据来自 AniList 公开 API；本站不存储番剧视频。
            </p>
          </div>
          <div class="flex flex-wrap items-center gap-2 text-sm">
            <span class="inline-flex items-center gap-2 rounded-md border border-zinc-200 bg-zinc-50 px-3 py-2 font-medium text-zinc-700">
              <Icon name="calendar" size="sm" />
              {{ weekRange.label }}
            </span>
            <button
              type="button"
              class="inline-flex items-center gap-2 rounded-md bg-zinc-950 px-3 py-2 font-semibold text-white transition hover:bg-zinc-800 disabled:cursor-wait disabled:opacity-70"
              :disabled="loadingState === 'loading'"
              @click="fetchAnimeData"
            >
              <Icon name="refresh" size="sm" />
              {{ statusText }}
            </button>
          </div>
        </div>

        <div class="grid grid-cols-7 gap-1 overflow-hidden rounded-lg border border-zinc-200 bg-zinc-100 p-1">
          <button
            v-for="(label, index) in dayLabels"
            :key="label"
            type="button"
            class="min-h-14 rounded-md px-2 py-2 text-center transition"
            :class="selectedDay === index ? 'bg-white text-zinc-950 shadow-sm' : 'text-zinc-500 hover:bg-white/70 hover:text-zinc-900'"
            @click="selectDay(index)"
          >
            <span class="block text-xs font-semibold uppercase">{{ label }}</span>
            <span class="mt-1 block text-[11px] sm:text-xs">{{ groupedSchedule[index]?.length || 0 }} 部</span>
          </button>
        </div>
      </div>
    </section>

    <section class="mx-auto grid max-w-7xl gap-6 px-4 py-6 sm:px-6 lg:grid-cols-[minmax(0,1.35fr)_420px] lg:px-8">
      <div class="space-y-6">
        <section class="rounded-lg border border-zinc-200 bg-white p-4 shadow-sm">
          <div class="mb-4 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 class="text-xl font-semibold tracking-normal">{{ selectedDayName }}放送</h2>
              <p class="text-sm text-zinc-500">按本地时间展示；实际上线以版权方平台为准。</p>
            </div>
            <span class="text-xs font-medium text-zinc-500">更新 {{ lastUpdated || '待同步' }}</span>
          </div>

          <div v-if="loadingState === 'loading'" class="grid gap-3">
            <div v-for="index in 4" :key="index" class="h-28 animate-pulse rounded-md bg-zinc-100"></div>
          </div>

          <div v-else-if="selectedSchedule.length" class="grid gap-3">
            <button
              v-for="item in selectedSchedule"
              :key="item.id"
              type="button"
              class="group grid grid-cols-[78px_minmax(0,1fr)] gap-3 rounded-md border p-2 text-left transition sm:grid-cols-[96px_minmax(0,1fr)_88px]"
              :class="featuredAnime?.id === item.id ? 'border-zinc-950 bg-zinc-50' : 'border-zinc-200 bg-white hover:border-zinc-300 hover:bg-zinc-50'"
              @click="selectAnime(item)"
            >
              <img :src="item.cover" :alt="item.title" class="h-28 w-full rounded object-cover sm:h-32">
              <span class="min-w-0 py-1">
                <span class="flex flex-wrap items-center gap-2">
                  <span class="rounded bg-zinc-950 px-2 py-1 text-xs font-semibold text-white">{{ item.timeLabel }}</span>
                  <span class="rounded bg-zinc-100 px-2 py-1 text-xs font-semibold text-zinc-700">{{ item.episodeLabel }}</span>
                  <span class="rounded bg-zinc-100 px-2 py-1 text-xs font-medium text-zinc-600">{{ item.format }}</span>
                </span>
                <span class="mt-3 block truncate text-lg font-semibold tracking-normal text-zinc-950">{{ item.title }}</span>
                <span class="mt-1 block truncate text-sm text-zinc-500">{{ item.nativeTitle }}</span>
                <span class="mt-3 flex flex-wrap gap-2">
                  <span v-for="genre in item.genres" :key="genre" class="rounded border border-zinc-200 px-2 py-1 text-xs text-zinc-600">{{ genre }}</span>
                </span>
              </span>
              <span class="hidden flex-col items-end justify-center gap-2 text-sm text-zinc-500 sm:flex">
                <span class="font-semibold text-zinc-950">{{ item.score }}</span>
                <span>{{ item.popularity }} 热度</span>
              </span>
            </button>
          </div>

          <div v-else class="rounded-md border border-dashed border-zinc-300 px-4 py-12 text-center">
            <h3 class="text-base font-semibold">当天暂无公开放送条目</h3>
            <p class="mt-2 text-sm text-zinc-500">可以切换日期，或查看本季热门条目。</p>
          </div>
        </section>

        <section class="rounded-lg border border-zinc-200 bg-white p-4 shadow-sm">
          <div class="mb-4 flex items-center justify-between gap-3">
            <div>
              <h2 class="text-xl font-semibold tracking-normal">本季热门</h2>
              <p class="text-sm text-zinc-500">来自 AniList trending 排序。</p>
            </div>
            <Icon name="trendingUp" class="text-zinc-500" />
          </div>
          <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-4">
            <a
              v-for="item in trending"
              :key="item.id"
              :href="item.siteUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="group overflow-hidden rounded-md border border-zinc-200 bg-white transition hover:-translate-y-0.5 hover:border-zinc-300 hover:shadow-card-hover"
            >
              <img :src="item.cover" :alt="item.title" class="aspect-[3/4] w-full object-cover">
              <span class="block space-y-1 px-3 py-3">
                <span class="block truncate text-sm font-semibold text-zinc-950">{{ item.title }}</span>
                <span class="block text-xs text-zinc-500">{{ item.timeLabel }} · {{ item.score }}</span>
              </span>
            </a>
          </div>
        </section>
      </div>

      <aside class="space-y-6">
        <section class="overflow-hidden rounded-lg border border-zinc-200 bg-white shadow-sm">
          <div class="relative min-h-48">
            <img :src="featuredAnime?.banner || fallbackBanner" :alt="featuredAnime?.title || 'Anime banner'" class="h-56 w-full object-cover">
            <div class="absolute inset-0 bg-gradient-to-t from-black/70 via-black/10 to-transparent"></div>
            <div class="absolute bottom-0 left-0 right-0 p-4 text-white">
              <p class="text-xs font-semibold uppercase tracking-normal text-white/75">Selected title</p>
              <h2 class="mt-1 line-clamp-2 text-2xl font-semibold tracking-normal">{{ featuredAnime?.title }}</h2>
            </div>
          </div>
          <div class="space-y-4 p-4">
            <div class="grid grid-cols-3 gap-2 text-center">
              <div class="rounded-md bg-zinc-50 px-2 py-3">
                <p class="text-xs text-zinc-500">Score</p>
                <p class="mt-1 font-semibold">{{ featuredAnime?.score }}</p>
              </div>
              <div class="rounded-md bg-zinc-50 px-2 py-3">
                <p class="text-xs text-zinc-500">Heat</p>
                <p class="mt-1 font-semibold">{{ featuredAnime?.popularity }}</p>
              </div>
              <div class="rounded-md bg-zinc-50 px-2 py-3">
                <p class="text-xs text-zinc-500">Type</p>
                <p class="mt-1 truncate font-semibold">{{ featuredAnime?.format }}</p>
              </div>
            </div>
            <div class="flex flex-wrap gap-2">
              <a
                v-for="link in watchLinks"
                :key="link.label"
                :href="link.href"
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex items-center gap-1 rounded-md px-3 py-2 text-xs font-semibold ring-1 transition hover:brightness-95"
                :class="link.tone"
              >
                {{ link.label }}
                <Icon name="externalLink" size="xs" />
              </a>
            </div>
          </div>
        </section>

        <section class="rounded-lg border border-zinc-200 bg-white p-4 shadow-sm">
          <div class="mb-3 flex items-center justify-between gap-3">
            <div>
              <h2 class="text-lg font-semibold tracking-normal">开放许可示例播放</h2>
              <p class="text-sm text-zinc-500">Sintel trailer, Blender Foundation / W3C media sample.</p>
            </div>
            <Icon name="play" class="text-zinc-500" />
          </div>
          <video
            class="aspect-video w-full rounded-md bg-black"
            controls
            preload="metadata"
            :poster="sampleVideoPoster"
          >
            <source :src="sampleVideoSrc" type="video/mp4">
          </video>
        </section>

        <section class="rounded-lg border border-zinc-200 bg-white p-4 shadow-sm">
          <h2 class="text-lg font-semibold tracking-normal">来源策略</h2>
          <div class="mt-3 grid gap-3 text-sm text-zinc-600">
            <div class="flex gap-3 rounded-md bg-zinc-50 p-3">
              <Icon name="database" size="sm" class="mt-0.5 flex-shrink-0 text-zinc-500" />
              <span>AniList GraphQL 提供公开番剧、播出时间和封面数据。</span>
            </div>
            <div class="flex gap-3 rounded-md bg-zinc-50 p-3">
              <Icon name="link" size="sm" class="mt-0.5 flex-shrink-0 text-zinc-500" />
              <span>观看入口优先使用 AniList 标注的官方 streaming 外链，并按平台 allowlist 过滤；没有直达入口时降级到官方平台搜索。</span>
            </div>
            <div class="flex gap-3 rounded-md bg-zinc-50 p-3">
              <Icon name="shield" size="sm" class="mt-0.5 flex-shrink-0 text-zinc-500" />
              <span>本站不解析隐藏视频地址，不代理、不下载、不缓存第三方番剧视频。</span>
            </div>
            <div v-if="loadError" class="rounded-md border border-amber-200 bg-amber-50 p-3 text-amber-800">
              AniList 暂不可用：{{ loadError }}
            </div>
          </div>
        </section>
      </aside>
    </section>
  </main>
</template>
