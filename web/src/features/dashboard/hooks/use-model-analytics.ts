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
import { skipToken, useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'

import { getModelAnalytics } from '@/features/dashboard/api'
import type {
  DashboardFilters,
  ModelAnalyticsQuery,
} from '@/features/dashboard/types'

export function useModelAnalytics(filters: DashboardFilters, enabled: boolean) {
  const query = useMemo<ModelAnalyticsQuery | null>(() => {
    if (!filters.start_timestamp || !filters.end_timestamp) return null
    const start = Math.floor(filters.start_timestamp.getTime() / 1000)
    const end = Math.floor(filters.end_timestamp.getTime() / 1000)
    if (end < start) return null
    return {
      start_timestamp: start,
      end_timestamp: end,
      granularity: filters.time_granularity,
      username: filters.username?.trim() || undefined,
      models: filters.models?.length ? [...filters.models].sort() : undefined,
    }
  }, [filters])

  const analyticsQuery = useQuery({
    queryKey: [
      'model-analytics',
      query?.start_timestamp,
      query?.end_timestamp,
      query?.granularity,
      query?.username,
      query?.models,
    ],
    queryFn: query ? () => getModelAnalytics(query) : skipToken,
    enabled: enabled && query != null,
    refetchInterval: 60_000,
    retry: false,
  })
  return {
    analyticsQuery,
    validRange: query != null,
  }
}
