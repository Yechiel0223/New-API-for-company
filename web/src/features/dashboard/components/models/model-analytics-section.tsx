/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { useModelAnalytics } from '@/features/dashboard/hooks/use-model-analytics'
import {
  buildDefaultDashboardFilters,
  getSavedChartPreferences,
} from '@/features/dashboard/lib/filters'
import type { DashboardFilters } from '@/features/dashboard/types'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { ConsumptionDistributionChart } from './consumption-distribution-chart'
import { LogStatCards } from './log-stat-cards'
import { ModelAnalyticsToolbar } from './model-analytics-toolbar'
import { ModelCharts } from './model-charts'

export function ModelAnalyticsSection(props: { username?: string }) {
  const { t } = useTranslation()
  const role = useAuthStore((state) => state.auth.user?.role)
  const isAdmin = Boolean(role && role >= ROLE.ADMIN)
  const preferences = useMemo(() => getSavedChartPreferences(), [])
  const [filters, setFilters] = useState<DashboardFilters>(() =>
    buildDefaultDashboardFilters(preferences)
  )
  const scopedFilters = useMemo(
    () => ({ ...filters, username: props.username }),
    [filters, props.username]
  )
  const analytics = useModelAnalytics(scopedFilters, isAdmin)
  const analyticsData = analytics.analyticsQuery.isError
    ? undefined
    : analytics.analyticsQuery.data
  const refresh = () => {
    void analytics.analyticsQuery.refetch()
  }

  if (!isAdmin) {
    return (
      <div className='text-muted-foreground rounded-lg border p-6 text-center text-sm'>
        {t('Model analytics is available to administrators.')}
      </div>
    )
  }

  return (
    <div className='space-y-3 sm:space-y-4'>
      <ModelAnalyticsToolbar
        filters={scopedFilters}
        availableModels={analyticsData?.available_models ?? []}
        onFiltersChange={setFilters}
        onRefresh={refresh}
        refreshing={analytics.analyticsQuery.isFetching}
      />
      <LogStatCards
        analytics={analyticsData}
        loading={analytics.analyticsQuery.isPending}
        error={analytics.analyticsQuery.isError}
        onRetry={refresh}
      />
      <ConsumptionDistributionChart
        analytics={analyticsData}
        loading={analytics.analyticsQuery.isPending}
        error={analytics.analyticsQuery.isError}
        onRetry={refresh}
        defaultChartType={preferences.consumptionDistributionChart}
      />
      <ModelCharts
        analytics={analyticsData}
        loading={analytics.analyticsQuery.isPending}
        error={analytics.analyticsQuery.isError}
        onRetry={refresh}
        defaultChartTab={preferences.modelAnalyticsChart}
      />
    </div>
  )
}
