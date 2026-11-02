<template>
  <div class="calendar-page page">
    <div class="page-head cal-head">
      <v-menu :close-on-content-click="false" location="bottom end">
        <template #activator="{ props: p }">
          <v-btn v-bind="p" variant="tonal" size="small" :prepend-icon="mdiFilterVariant">
            {{ hidden.length ? `${t('calendar.filter')} (${visibleCount}/${todo.cards.length})` : t('calendar.allCards') }}
          </v-btn>
        </template>
        <v-list min-width="220" density="compact">
          <v-list-item v-for="c in todo.sortedCards" :key="c.id" @click="toggleCard(c.id)">
            <template #prepend>
              <v-checkbox-btn :model-value="!hidden.includes(c.id)" :color="cardColor(c)" density="compact" />
            </template>
            <v-list-item-title>
              <span class="dot" :style="{ background: cardColor(c) }" />
              {{ c.name }}
            </v-list-item-title>
          </v-list-item>
        </v-list>
      </v-menu>
      <v-switch
        :model-value="settings.s.calendar_show_completed"
        :label="t('calendar.showCompleted')"
        color="primary"
        density="compact"
        hide-details
        inset
        class="flex-grow-0"
        @update:model-value="(v: unknown) => settings.set({ calendar_show_completed: !!v })"
      />
    </div>

    <!-- Calendar navigation, shared by the month, week and day views. -->
    <div class="cal-nav">
      <v-btn icon variant="text" size="small" :aria-label="t('calendar.prev')" @click="api()?.prev()">
        <v-icon :icon="mdiChevronLeft" />
      </v-btn>
      <v-menu v-model="picker" :close-on-content-click="false" location="bottom start">
        <template #activator="{ props: p }">
          <v-btn v-bind="p" variant="text" class="cal-title" :aria-label="t('calendar.pickDate')">
            <span class="cal-title-text">{{ nav.title }}</span>
            <v-icon :icon="mdiMenuDown" size="18" class="ml-1" />
          </v-btn>
        </template>
        <v-date-picker
          v-if="picker"
          :model-value="pickerValue"
          :first-day-of-week="settings.s.week_start"
          color="primary"
          control-variant="modal"
          class="cal-picker"
          hide-header
          show-adjacent-months
          @update:model-value="pickDate"
        />
      </v-menu>
      <v-btn icon variant="text" size="small" :aria-label="t('calendar.next')" @click="api()?.next()">
        <v-icon :icon="mdiChevronRight" />
      </v-btn>
      <v-spacer />
      <v-btn variant="text" size="small" class="px-2" @click="api()?.today()">{{ t('common.today') }}</v-btn>
      <v-btn-toggle
        v-if="!mobile"
        :model-value="nav.view"
        mandatory
        divided
        density="compact"
        variant="outlined"
        class="cal-views ml-1"
        @update:model-value="changeView"
      >
        <v-btn v-for="v in viewButtons" :key="v.value" :value="v.value" size="small">{{ v.title }}</v-btn>
      </v-btn-toggle>
    </div>

    <div class="cal-wrap">
      <FullCalendar ref="calRef" :options="options" />
    </div>

    <div v-if="mobile" class="cal-nav cal-nav-bottom">
      <v-btn-toggle
        :model-value="nav.view"
        mandatory
        divided
        density="compact"
        variant="outlined"
        class="cal-views"
        @update:model-value="changeView"
      >
        <v-btn v-for="v in viewButtons" :key="v.value" :value="v.value" size="small">{{ v.title }}</v-btn>
      </v-btn-toggle>
    </div>

    <v-dialog v-model="quick.open" max-width="420">
      <v-card :title="t('calendar.quickCreate')" :subtitle="quickLabel">
        <v-card-text>
          <v-text-field
            v-model="quick.title"
            :label="t('item.title')"
            variant="outlined"
            autofocus
            maxlength="500"
            class="mb-3"
            hide-details
            @keydown.enter="createQuick"
          />
          <v-select v-model="quick.cardId" :items="cardOptions" :label="t('item.card')" variant="outlined" hide-details />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="quick.open = false">{{ t('common.cancel') }}</v-btn>
          <v-btn color="primary" variant="flat" :disabled="!quick.title.trim() || !quick.cardId" @click="createQuick">
            {{ t('common.create') }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDisplay } from 'vuetify'
import FullCalendar from '@fullcalendar/vue3'
import dayGridPlugin from '@fullcalendar/daygrid'
import timeGridPlugin from '@fullcalendar/timegrid'
import interactionPlugin, { type DateClickArg } from '@fullcalendar/interaction'
import allLocales from '@fullcalendar/core/locales-all'
import type {
  CalendarOptions,
  DatesSetArg,
  DayCellContentArg,
  DayHeaderContentArg,
  EventClickArg,
  EventDropArg,
  EventInput,
  MoreLinkContentArg,
} from '@fullcalendar/core'
import { mdiChevronLeft, mdiChevronRight, mdiFilterVariant, mdiMenuDown } from '@mdi/js'
import { useSettingsStore } from '@/stores/settings'
import { useTodoStore } from '@/stores/todo'
import { useUiStore } from '@/stores/ui'
import { cardColor } from '@/utils/cards'
import { dayjs, fmtDate, getTz, isOverdue, localDue, nowTz, timePattern, toStoredDue } from '@/utils/time'

const { t, locale } = useI18n()
const { mobile } = useDisplay()
const todo = useTodoStore()
const settings = useSettingsStore()
const ui = useUiStore()

const hidden = computed(() => settings.s.calendar_hidden_cards ?? [])
const visibleCount = computed(() => todo.cards.filter((c) => !hidden.value.includes(c.id)).length)

function toggleCard(id: number) {
  const set = new Set(hidden.value)
  if (set.has(id)) set.delete(id)
  else set.add(id)
  settings.set({ calendar_hidden_cards: [...set] })
}

const OVERDUE = '#E53935'
const DONE = '#9E9E9E'

/*
 * FullCalendar runs in UTC mode and receives "wall clock" times of the user's
 * timezone, so the calendar shows the user's chosen timezone without a
 * timezone plugin. Dates coming back from FullCalendar are read the same way.
 */
const wall = (d: Date) => dayjs.utc(d)

const events = computed<EventInput[]>(() => {
  const out: EventInput[] = []
  const showDone = settings.s.calendar_show_completed
  for (const it of todo.items) {
    if (hidden.value.includes(it.card_id)) continue
    const card = todo.cardById(it.card_id)
    if (!card) continue
    if (it.completed) {
      if (!showDone || !it.completed_at) continue
      out.push({
        id: String(it.id),
        title: '✓ ' + it.title,
        start: dayjs(it.completed_at).tz(getTz()).format('YYYY-MM-DDTHH:mm:ss'),
        allDay: false,
        editable: false,
        backgroundColor: DONE,
        borderColor: DONE,
        textColor: '#fff',
        classNames: ['ev-done'],
      })
      continue
    }
    const due = localDue(it)
    if (!due) continue
    const color = isOverdue(it) ? OVERDUE : cardColor(card)
    out.push({
      id: String(it.id),
      title: it.title,
      start: due.time ? `${due.date}T${due.time}:00` : due.date,
      allDay: !due.time,
      backgroundColor: color,
      borderColor: color,
      textColor: '#fff',
    })
  }
  return out
})

// Kept stable so FullCalendar does not re-initialise when other options change.
const plugins = [dayGridPlugin, timeGridPlugin, interactionPlugin]
const wallNow = () => nowTz().format('YYYY-MM-DDTHH:mm:ss')

const fcLocale = computed(() => locale.value.toLowerCase())
const hour12 = computed(() => settings.s.time_format === '12h')

/*
 * The toolbar is our own Vuetify row instead of FullCalendar's, so the title can open a date picker.
 * FullCalendar still owns the navigation state; datesSet mirrors it into `nav`.
 */
const calRef = ref<InstanceType<typeof FullCalendar> | null>(null)
const api = () => calRef.value?.getApi()
const nav = reactive({ view: 'dayGridMonth', title: '', date: '' })
const picker = ref(false)

const viewButtons = computed(() => [
  { value: 'dayGridMonth', title: t('calendar.month') },
  { value: 'timeGridWeek', title: t('calendar.week') },
  { value: 'timeGridDay', title: t('calendar.day') },
])

function onDatesSet(arg: DatesSetArg) {
  nav.view = arg.view.type
  nav.title = arg.view.title
  nav.date = wall(arg.view.calendar.getDate()).format('YYYY-MM-DD')
}

function changeView(v: unknown) {
  if (typeof v === 'string' && v !== nav.view) api()?.changeView(v)
}

// v-date-picker works with local Date objects; the calendar works with wall-clock date strings.
const pickerValue = computed(() => (nav.date ? dayjs(nav.date).toDate() : undefined))

function pickDate(v: unknown) {
  const d = Array.isArray(v) ? v[0] : v
  if (!d) return
  api()?.gotoDate(dayjs(d as Date).format('YYYY-MM-DD'))
  picker.value = false
}

const escapeHtml = (s: string) =>
  s.replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c] as string)

// Week and day column headers: small gray weekday above the day number, like the month view's numbers.
const timeGridHeader = (arg: DayHeaderContentArg) => ({
  html: `<span class="dh-wd">${escapeHtml(arg.text)}</span><span class="dh-num">${arg.date.getUTCDate()}</span>`,
})

const options = computed<CalendarOptions>(() => ({
  plugins,
  initialView: 'dayGridMonth',
  timeZone: 'UTC',
  now: wallNow,
  locales: allLocales,
  locale: fcLocale.value,
  firstDay: settings.s.week_start,
  height: '100%',
  headerToolbar: false,
  allDayText: '',
  eventTimeFormat: { hour: hour12.value ? 'numeric' : '2-digit', minute: '2-digit', hour12: hour12.value },
  slotLabelFormat: { hour: hour12.value ? 'numeric' : '2-digit', minute: '2-digit', hour12: hour12.value },
  scrollTime: '08:00:00',
  nowIndicator: true,
  dayMaxEvents: true,
  views: {
    dayGridMonth: {
      // iOS-style month grid: single-character weekdays (日 一 … 六) and bare day numbers.
      // The zh locale renders "26日", which wraps in narrow columns and stretches rows unevenly.
      dayHeaderFormat: { weekday: 'narrow' },
      dayCellContent: (arg: DayCellContentArg) => String(arg.date.getUTCDate()),
      moreLinkContent: (arg: MoreLinkContentArg) => `+${arg.num}`,
    },
    timeGrid: {
      dayHeaderContent: timeGridHeader,
      // The all-day row grows with its content, so cap it and let "+N" open the popover.
      dayMaxEvents: mobile.value ? 2 : 3,
      moreLinkContent: (arg: MoreLinkContentArg) => `+${arg.num}`,
      eventMinHeight: 20,
    },
    timeGridWeek: {
      dayHeaderFormat: { weekday: 'narrow' },
      titleFormat: { year: 'numeric', month: 'long' },
    },
    timeGridDay: {
      dayHeaderFormat: { weekday: 'short' },
    },
  },
  eventDisplay: 'block',
  editable: true,
  eventDurationEditable: false,
  events: events.value,
  eventClick: onEventClick,
  eventDrop: onEventDrop,
  dateClick: onDateClick,
  datesSet: onDatesSet,
}))

function onEventClick(info: EventClickArg) {
  info.jsEvent.preventDefault()
  ui.openDetail(Number(info.event.id))
}

function onEventDrop(info: EventDropArg) {
  const it = todo.itemById(Number(info.event.id))
  const start = info.event.start
  if (!it || !start || it.completed) {
    info.revert()
    return
  }
  const date = wall(start).format('YYYY-MM-DD')
  let time: string | null
  if (info.view.type.startsWith('dayGrid')) {
    // Month view only changes the day and keeps the existing time.
    time = localDue(it)?.time ?? null
  } else {
    time = info.event.allDay ? null : wall(start).format('HH:mm')
  }
  todo.updateItem(it.id, toStoredDue(date, time))
}

const quick = reactive({ open: false, title: '', cardId: null as number | null, date: '', time: null as string | null })

const quickLabel = computed(() => {
  if (!quick.date) return ''
  const d = fmtDate(dayjs(quick.date))
  return quick.time ? `${d} ${dayjs(`2000-01-01T${quick.time}`).format(timePattern())}` : d
})

const cardOptions = computed(() => todo.sortedCards.map((c) => ({ title: c.name, value: c.id })))

function onDateClick(info: DateClickArg) {
  if (!todo.cards.length) {
    ui.notify(t('quadrant.noCard'))
    return
  }
  const w = wall(info.date)
  quick.date = w.format('YYYY-MM-DD')
  quick.time = info.allDay ? null : w.format('HH:mm')
  quick.title = ''
  quick.cardId = todo.currentCardId
  quick.open = true
}

async function createQuick() {
  const title = quick.title.trim()
  if (!title || !quick.cardId) return
  try {
    await todo.createItem({ card_id: quick.cardId, title, ...toStoredDue(quick.date, quick.time) })
    quick.open = false
  } catch (e) {
    ui.error(e)
  }
}
</script>

<style scoped>
.calendar-page {
  display: flex;
  flex-direction: column;
  height: calc(100dvh - var(--v-layout-top, 0px) - var(--v-layout-bottom, 0px));
}
.cal-head {
  margin-bottom: 4px;
}
.cal-nav {
  display: flex;
  align-items: center;
  gap: 2px;
  min-height: 44px;
  flex: 0 0 auto;
}
.cal-nav-bottom {
  justify-content: center;
  min-height: 0;
  padding-top: 8px;
}
.cal-title {
  min-width: 0;
  padding: 0 6px;
  font-size: 17px;
  font-weight: 600;
  letter-spacing: 0;
  text-transform: none;
}
.cal-title :deep(.v-btn__content) {
  min-width: 0;
}
.cal-title-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cal-views .v-btn {
  min-width: 44px;
  letter-spacing: 0;
}
.cal-wrap {
  flex: 1 1 auto;
  min-height: 420px;
}
.dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  margin-right: 6px;
}
.cal-wrap :deep(.fc) {
  --fc-border-color: rgba(var(--v-border-color), var(--v-border-opacity));
  --fc-page-bg-color: rgb(var(--v-theme-background));
  --fc-neutral-bg-color: rgba(var(--v-theme-on-surface), 0.04);
  --fc-today-bg-color: rgba(var(--v-theme-primary), 0.08);
  --fc-button-bg-color: rgb(var(--v-theme-primary));
  --fc-button-border-color: rgb(var(--v-theme-primary));
  --fc-button-hover-bg-color: rgba(var(--v-theme-primary), 0.85);
  --fc-button-hover-border-color: rgba(var(--v-theme-primary), 0.85);
  --fc-button-active-bg-color: rgba(var(--v-theme-primary), 0.7);
  --fc-button-active-border-color: rgba(var(--v-theme-primary), 0.7);
  --fc-now-indicator-color: rgb(var(--v-theme-error));
}
.cal-wrap :deep(.fc-event) {
  cursor: pointer;
}
.cal-wrap :deep(.ev-done .fc-event-title) {
  text-decoration: line-through;
}
.cal-wrap :deep(.fc a) {
  color: inherit;
  text-decoration: none;
}
/* ---------- Month view, iOS style: row hairlines only, no vertical lines or outer frame ---------- */
.cal-wrap :deep(.fc) {
  --cal-sep: rgba(60, 60, 67, 0.29); /* iOS separator */
  --cal-weekday: #6e6e73; /* iOS gray, darker than #8E8E93 so 13px text stays readable on white */
}
.v-theme--dark .cal-wrap :deep(.fc) {
  --cal-sep: rgba(84, 84, 88, 0.6);
  --cal-weekday: #8e8e93;
}
.cal-wrap :deep(.fc-dayGridMonth-view .fc-scrollgrid),
.cal-wrap :deep(.fc-dayGridMonth-view td),
.cal-wrap :deep(.fc-dayGridMonth-view th) {
  border: 0;
}
/* The first week's top border is hidden by FullCalendar's nested table, so the header draws that line. */
.cal-wrap :deep(.fc-dayGridMonth-view .fc-col-header-cell),
.cal-wrap :deep(.fc-dayGridMonth-view .fc-daygrid-day) {
  border-top: 0.5px solid var(--cal-sep);
}
.cal-wrap :deep(.fc-dayGridMonth-view .fc-col-header-cell) {
  border-top: 0;
  border-bottom: 0.5px solid var(--cal-sep);
}
.cal-wrap :deep(.fc-dayGridMonth-view .fc-col-header-cell-cushion) {
  padding: 4px 0 8px;
  font-size: 13px;
  font-weight: 400;
  line-height: 1;
  color: var(--cal-weekday);
}
.cal-wrap :deep(.fc-dayGridMonth-view .fc-daygrid-day.fc-day-today) {
  background: none;
}
.cal-wrap :deep(.fc-dayGridMonth-view .fc-daygrid-day-top) {
  justify-content: center;
  padding-top: 6px;
}
.cal-wrap :deep(.fc-dayGridMonth-view .fc-daygrid-day-number) {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  border-radius: 50%;
  font-size: 16px;
  font-weight: 500;
  line-height: 1;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}
.cal-wrap :deep(.fc-dayGridMonth-view .fc-day-today .fc-daygrid-day-number) {
  color: rgb(var(--v-theme-on-primary));
  background: rgb(var(--v-theme-primary));
  font-weight: 600;
}
.cal-wrap :deep(.fc-dayGridMonth-view .fc-daygrid-day-events) {
  margin-top: 4px;
}
.cal-wrap :deep(.fc-dayGridMonth-view .fc-daygrid-more-link) {
  font-size: 11px;
  white-space: nowrap;
}
/* ---------- Week and day views: same rules as the month view (hairline rows, no vertical lines or frame) ---------- */
.cal-wrap :deep(.fc-timegrid .fc-scrollgrid),
.cal-wrap :deep(.fc-timegrid td),
.cal-wrap :deep(.fc-timegrid th) {
  border: 0;
}
.cal-wrap :deep(.fc-timegrid .fc-col-header-cell) {
  border-bottom: 0.5px solid var(--cal-sep);
}
.cal-wrap :deep(.fc-timegrid .fc-col-header-cell-cushion) {
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 4px 0 8px;
  font-weight: 400;
  line-height: 1;
}
.cal-wrap :deep(.fc-timegrid .dh-wd) {
  font-size: 13px;
  color: var(--cal-weekday);
}
.cal-wrap :deep(.fc-timegrid .dh-num) {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  font-size: 16px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}
.cal-wrap :deep(.fc-timegrid .fc-day-today .dh-num) {
  color: rgb(var(--v-theme-on-primary));
  background: rgb(var(--v-theme-primary));
  font-weight: 600;
}
.cal-wrap :deep(.fc-timegrid .fc-day-today) {
  background: none;
}
/* All-day row: a hairline below it instead of FullCalendar's shaded divider bar. */
.cal-wrap :deep(.fc-timegrid .fc-timegrid-divider) {
  padding: 0;
  background: none;
  border-bottom: 0.5px solid var(--cal-sep);
}
.cal-wrap :deep(.fc-timegrid .fc-daygrid-body .fc-daygrid-day-frame) {
  min-height: 28px;
  padding: 2px 0;
}
/* Only full hours get a line; half-hour slots stay blank. */
.cal-wrap :deep(.fc-timegrid .fc-timegrid-slot) {
  height: 1.6em;
}
.cal-wrap :deep(.fc-timegrid .fc-timegrid-slot-lane:not(.fc-timegrid-slot-minor)) {
  border-top: 0.5px solid var(--cal-sep);
}
.cal-wrap :deep(.fc-timegrid .fc-timegrid-slot-label-cushion) {
  padding: 0 6px;
  font-size: 11px;
  color: var(--cal-weekday);
  font-variant-numeric: tabular-nums;
}
.cal-wrap :deep(.fc-timegrid .fc-event) {
  font-size: 12px;
  border-radius: 4px;
}
.cal-wrap :deep(.fc-timegrid .fc-timegrid-event .fc-event-main) {
  padding: 2px 4px;
}
.cal-wrap :deep(.fc-timegrid .fc-event-title),
.cal-wrap :deep(.fc-timegrid .fc-event-time) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cal-wrap :deep(.fc-timegrid .fc-daygrid-more-link) {
  font-size: 11px;
  white-space: nowrap;
}
.cal-wrap :deep(.fc-popover) {
  background: rgb(var(--v-theme-surface));
}
@media (max-width: 599px) {
  /* The picker is 328px wide by default, too wide for 320px phones. */
  .cal-picker {
    width: 296px;
  }
  .cal-picker :deep(.v-date-picker-month) {
    padding: 0 8px 8px;
  }
  .cal-picker :deep(.v-date-picker-month__days) {
    column-gap: 0;
  }
  .cal-picker :deep(.v-date-picker-controls) {
    padding: 4px 8px;
  }
  .cal-wrap :deep(.fc-dayGridMonth-view .fc-daygrid-event) {
    font-size: 11px;
  }
  .cal-wrap :deep(.fc-timegrid .fc-event) {
    font-size: 11px;
  }
  .cal-wrap :deep(.fc-timegrid .fc-timegrid-event .fc-event-main) {
    padding: 1px 2px;
  }
}
</style>
