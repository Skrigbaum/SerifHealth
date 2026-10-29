<script setup>
import { computed, onMounted, ref } from 'vue'

const API_BASE = 'http://localhost:8080'
const billingCode = ref('')
const billingCodeType = ref('')
const planId = ref('')
const providerIdentifier = ref('')
const rows = ref([])
const codeTypes = ref([])
const page = ref(1)
const pageSize = ref(50)
const totalCount = ref(0)
const loading = ref(false)
const error = ref('')
const sortKey = ref('negotiated_rate')
const sortAsc = ref(true)

const tableColumns = [
  { key: 'billing_code', label: 'Billing Code' },
  { key: 'billing_code_type', label: 'Code Type' },
  { key: 'plan_ids', label: 'Plan ID(s)' },
  { key: 'procedure', label: 'Procedure' },
  { key: 'tin_business_name', label: 'Provider' },
  { key: 'npi', label: 'NPI' },
  { key: 'tin_value', label: 'EIN / TIN' },
  { key: 'billing_class', label: 'Billing Class' },
  { key: 'setting', label: 'Setting' },
  { key: 'expiration_date', label: 'Expiration' },
  { key: 'negotiated_rate', label: 'Price' },
]

onMounted(async () => {
  await loadCodeTypes()
})

async function loadCodeTypes() {
  try {
    const result = await fetchApi(`${API_BASE}/api/search/billing-code-types`)
    codeTypes.value = result.types || []
  } catch (err) {
    console.error('Unable to load billing code types', err)
    error.value = errorMessage(err)
  }
}

async function fetchApi(url) {
  let response
  try {
    response = await fetch(url)
  } catch {
    throw new Error('Unable to connect to the API. Check that the backend is running.')
  }

  let result
  try {
    result = await response.json()
  } catch {
    throw new Error(`The API returned an invalid response (HTTP ${response.status}).`)
  }

  if (!response.ok) {
    throw new Error(result.error || `The API request failed (HTTP ${response.status}).`)
  }
  return result
}

function errorMessage(error) {
  return error instanceof Error ? error.message : 'The API request failed.'
}

async function search(targetPage = 1) {
  page.value = targetPage
  const code = billingCode.value.trim()
  if (!code) {
    error.value = 'Enter a billing code to search provider rates.'
    rows.value = []
    totalCount.value = 0
    return
  }

  loading.value = true
  error.value = ''
  rows.value = []

  try {
    const params = new URLSearchParams({ billing_code: code })
    params.set('page', String(page.value))
    params.set('sort_order', sortAsc.value ? 'asc' : 'desc')
    if (billingCodeType.value) params.set('billing_code_type', billingCodeType.value)
    if (planId.value.trim()) params.set('plan_id', planId.value.trim())
    if (providerIdentifier.value.trim()) params.set('provider_identifier', providerIdentifier.value.trim())

    const result = await fetchApi(`${API_BASE}/api/search/provider-price?${params}`)
    rows.value = result.items || []
    totalCount.value = result.total || 0
    pageSize.value = result.page_size || 50
  } catch (err) {
    console.error(err)
    error.value = errorMessage(err)
    rows.value = []
    totalCount.value = 0
  } finally {
    loading.value = false
  }
}

function sortBy(columnKey) {
  if (sortKey.value === columnKey) {
    sortAsc.value = !sortAsc.value
  } else {
    sortKey.value = columnKey
    sortAsc.value = true
  }
  if (columnKey === 'negotiated_rate') search(page.value)
}

function formatCell(row, key) {
  const value = row[key]
  if (value == null || value === '') return '—'
  if (key === 'negotiated_rate') return `$${Number(value).toFixed(2)}`
  if (key === 'expiration_date') {
    const [year, month, day] = String(value).split('-')
    if (year && month && day) return `${month}-${day}-${year}`
  }
  return value
}

const sortedRows = computed(() => {
  if (!sortKey.value || sortKey.value === 'negotiated_rate') return rows.value

  return [...rows.value].sort((left, right) => {
    const a = left[sortKey.value] ?? ''
    const b = right[sortKey.value] ?? ''

    const comparison = String(a).localeCompare(String(b), undefined, { numeric: true })
    return sortAsc.value ? comparison : -comparison
  })
})
const totalPages = computed(() => Math.max(1, Math.ceil(totalCount.value / pageSize.value)))
const firstResult = computed(() => (page.value - 1) * pageSize.value + 1)
const lastResult = computed(() => Math.min(page.value * pageSize.value, totalCount.value))
</script>

<template>
  <div class="page-shell">
    <div class="card">
      <header class="header">
        <div>
          <p class="eyebrow">SerifHealth</p>
          <h1>Rate Search</h1>
        </div>
      </header>

      <form class="search-panel" @submit.prevent="search()">
        <div class="field-group">
          <label for="billingCode">Billing Code</label>
          <input id="billingCode" v-model="billingCode" required placeholder="e.g. 10006" />
        </div>

        <div class="field-group">
          <label for="billingCodeType">Billing Code Type (optional)</label>
          <select id="billingCodeType" v-model="billingCodeType">
            <option value="">All types</option>
            <option v-for="option in codeTypes" :key="option" :value="option">
              {{ option }}
            </option>
          </select>
        </div>

        <div class="field-group">
          <label for="planId">Plan ID (optional)</label>
          <input id="planId" v-model="planId" placeholder="e.g. 25303NY005" />
        </div>

        <div class="field-group">
          <label for="providerIdentifier">Provider ID: NPI or EIN (optional)</label>
          <input id="providerIdentifier" v-model="providerIdentifier" placeholder="NPI or EIN / TIN" />
        </div>

        <button class="search-button" type="submit" :disabled="loading">
          {{ loading ? 'Searching...' : 'Search' }}
        </button>
      </form>

      <div v-if="error" class="error-banner" role="alert">{{ error }}</div>

      <div class="table-wrap">
        <table v-if="sortedRows.length">
          <thead>
            <tr>
              <th v-for="column in tableColumns" :key="column.key" :data-column="column.key">
                <button type="button" class="sort-button" @click="sortBy(column.key)">
                  {{ column.label }}
                  <span v-if="sortKey === column.key">{{ sortAsc ? '↑' : '↓' }}</span>
                </button>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in sortedRows" :key="row.id">
              <td v-for="column in tableColumns" :key="column.key" :data-column="column.key">
                <span
                  v-if="column.key === 'npi'"
                  class="clamped-cell-value"
                  :title="String(row.npi || '')"
                >
                  {{ formatCell(row, column.key) }}
                </span>
                <span
                  v-else-if="column.key === 'plan_ids'"
                  class="clamped-cell-value"
                  :title="String(row.plan_ids || '')"
                >
                  {{ formatCell(row, column.key) }}
                </span>
                <template v-else>
                  {{ formatCell(row, column.key) }}
                </template>
              </td>
            </tr>
          </tbody>
        </table>
        <div v-else class="empty-state">
          {{ loading ? 'Searching...' : error ? 'Search could not be completed.' : 'Enter a billing code to compare provider rates.' }}
        </div>
      </div>

      <nav v-if="totalCount" class="pagination" aria-label="Rate result pages">
        <span>Showing {{ firstResult }} to {{ lastResult }} of {{ totalCount }}</span>
        <div class="pagination-controls">
          <button type="button" @click="search(page - 1)" :disabled="loading || page <= 1">Previous</button>
          <span>Page {{ page }} of {{ totalPages }}</span>
          <button type="button" @click="search(page + 1)" :disabled="loading || page >= totalPages">Next</button>
        </div>
      </nav>
    </div>
  </div>
</template>
