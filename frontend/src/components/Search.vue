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