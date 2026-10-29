<script setup>
import { computed, onMounted, ref } from 'vue'

const API_BASE = 'http://localhost:8080'
const billingCode = ref('')
const billingCodeType = ref('')
const npi = ref('')
const tinValue = ref('')
const filterText = ref('')
const rows = ref([])
const codeTypes = ref([])
const loading = ref(false)
const error = ref('')
const sortKey = ref('name')
const sortAsc = ref(true)

const tableColumns = [
  { key: 'source', label: 'Source' },
  { key: 'billing_code', label: 'Billing Code' },
  { key: 'billing_code_type', label: 'Billing Code Type' },
  { key: 'name', label: 'Procedure / Provider' },
  { key: 'description', label: 'Description' },
  { key: 'npi', label: 'NPI' },
  { key: 'tin_value', label: 'TIN Value' },
  { key: 'tin_business_name', label: 'TIN Business Name' },
]

onMounted(async () => {
  await loadCodeTypes()
})

async function loadCodeTypes() {
  try {
    const response = await fetch(`${API_BASE}/api/search/billing-code-types`)
    const payload = await response.json()
    codeTypes.value = payload.types || []
  } catch (err) {
    console.error('Unable to load billing code types', err)
  }
}

async function search() {
  const hasBillingCode = billingCode.value.trim()
  const hasNpi = npi.value.trim()
  const hasTinValue = tinValue.value.trim()

  if (!hasBillingCode && !hasNpi && !hasTinValue) {
    error.value = 'Enter a billing code, NPI, or TIN value to search.'
    rows.value = []
    return
  }

  loading.value = true
  error.value = ''
  rows.value = []

  try {
    const requests = []

    if (hasBillingCode) {
      const params = new URLSearchParams({ billing_code: billingCode.value.trim() })
      if (billingCodeType.value) params.set('billing_code_type', billingCodeType.value)
      requests.push(fetch(`${API_BASE}/api/search/billing-code?${params.toString()}`).then((res) => res.json()))
    }

    if (hasNpi) {
      requests.push(fetch(`${API_BASE}/api/search/provider?npi=${encodeURIComponent(npi.value.trim())}`).then((res) => res.json()))
    }

    if (hasTinValue) {
      requests.push(fetch(`${API_BASE}/api/search/provider?tin_value=${encodeURIComponent(tinValue.value.trim())}`).then((res) => res.json()))
    }

    const results = await Promise.all(requests)
    const combined = []

    results.forEach((result) => {
      if (!result || !Array.isArray(result.items)) return

      if (result.billing_code) {
        result.items.forEach((item) => {
          combined.push({
            id: `${item.id}-network`,
            source: 'Network Item',
            billing_code: item.billing_code || '',
            billing_code_type: item.billing_code_type || '',
            name: item.name || '',
            description: item.description || '',
            npi: '',
            tin_value: '',
            tin_business_name: '',
          })
        })
        return
      }

      result.items.forEach((item) => {
        combined.push({
          id: `${item.id}-provider`,
          source: 'Provider',
          billing_code: '',
          billing_code_type: '',
          name: item.tin_business_name || item.tin_value || item.npi || 'Provider',
          description: item.tin_type || '',
          npi: item.npi || '',
          tin_value: item.tin_value || '',
          tin_business_name: item.tin_business_name || '',
        })
      })
    })

    rows.value = combined
  } catch (err) {
    console.error(err)
    error.value = 'Search failed. Check the API server and try again.'
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
}

const filteredRows = computed(() => {
  const query = filterText.value.trim().toLowerCase()

  const visible = query
    ? rows.value.filter((row) => {
        const haystack = [
          row.source,
          row.billing_code,
          row.billing_code_type,
          row.name,
          row.description,
          row.npi,
          row.tin_value,
          row.tin_business_name,
        ]
          .join(' ')
          .toLowerCase()
        return haystack.includes(query)
      })
    : rows.value

  return [...visible].sort((left, right) => {
    const a = left[sortKey.value] ?? ''
    const b = right[sortKey.value] ?? ''

    const comparison = String(a).localeCompare(String(b), undefined, { numeric: true })
    return sortAsc.value ? comparison : -comparison
  })
})
</script>

<template>
  <div class="page-shell">
    <div class="card">
      <header class="header">
        <div>
          <p class="eyebrow">SerifHealth</p>
          <h1>Network Search</h1>
        </div>
      </header>

      <section class="search-panel">
        <div class="field-group">
          <label for="billingCode">Billing Code</label>
          <input id="billingCode" v-model="billingCode" placeholder="e.g. 10006" />
        </div>

        <div class="field-group">
          <label for="billingCodeType">Billing Code Type</label>
          <select id="billingCodeType" v-model="billingCodeType">
            <option value="">All</option>
            <option v-for="option in codeTypes" :key="option" :value="option">
              {{ option }}
            </option>
          </select>
        </div>

        <div class="field-group">
          <label for="npi">NPI</label>
          <input id="npi" v-model="npi" placeholder="e.g. 1396747861" />
        </div>

        <div class="field-group">
          <label for="tinValue">TIN Value</label>
          <input id="tinValue" v-model="tinValue" placeholder="e.g. 11-3471596" />
        </div>

        <button class="search-button" @click="search" :disabled="loading">
          {{ loading ? 'Searching...' : 'Search' }}
        </button>
      </section>

      <div class="table-toolbar">
        <label class="filter-box">
          <span>Filter table</span>
          <input v-model="filterText" placeholder="Search results..." />
        </label>
      </div>

      <div v-if="error" class="error-banner">{{ error }}</div>

      <div class="table-wrap">
        <table v-if="filteredRows.length">
          <thead>
            <tr>
              <th v-for="column in tableColumns" :key="column.key">
                <button type="button" class="sort-button" @click="sortBy(column.key)">
                  {{ column.label }}
                  <span v-if="sortKey === column.key">{{ sortAsc ? '↑' : '↓' }}</span>
                </button>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in filteredRows" :key="row.id">
              <td>{{ row.source }}</td>
              <td>{{ row.billing_code || '—' }}</td>
              <td>{{ row.billing_code_type || '—' }}</td>
              <td>{{ row.name || '—' }}</td>
              <td>{{ row.description || '—' }}</td>
              <td>{{ row.npi || '—' }}</td>
              <td>{{ row.tin_value || '—' }}</td>
              <td>{{ row.tin_business_name || '—' }}</td>
            </tr>
          </tbody>
        </table>
        <div v-else class="empty-state">
          No rows match the current search.
        </div>
      </div>
    </div>
  </div>
</template>
