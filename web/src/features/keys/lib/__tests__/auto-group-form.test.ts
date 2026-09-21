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
import i18n from 'i18next'
import { describe, expect, it } from 'vitest'

import {
  getApiKeyFormSchema,
  transformFormDataToPayload,
} from '../api-key-form'

describe('unrestricted API key creation', () => {
  it('creates an unlimited permanent key in the default group', () => {
    expect(transformFormDataToPayload({ name: 'demo' })).toEqual({
      name: 'demo',
      remain_quota: 0,
      unlimited_quota: true,
      group: 'default',
      auto_groups: [],
      cross_group_retry: false,
      expired_time: -1,
      model_limits_enabled: false,
      model_limits: '',
      allow_ips: '',
    })
  })
  it.each(['', '   '])('rejects an empty key name %j', (name) => {
    expect(getApiKeyFormSchema(i18n.t).safeParse({ name }).success).toBe(false)
  })
})
