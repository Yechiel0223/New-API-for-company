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
import { describe, expect, it } from 'vitest'

import type { ApiKey } from '../../types'
import {
  getApiKeyFormDefaultValues,
  transformApiKeyToFormDefaults,
  transformFormDataToPayload,
} from '../api-key-form'

describe('default API key configuration', () => {
  it('uses the default group even when automatic grouping is enabled globally', () => {
    const payload = transformFormDataToPayload(getApiKeyFormDefaultValues(true))
    expect(payload).toMatchObject({
      group: 'default',
      auto_groups: [],
      cross_group_retry: false,
      expired_time: -1,
      model_limits_enabled: false,
      model_limits: '',
      allow_ips: '',
    })
  })
  it('clears removed restrictions on edit without changing quota or the key name', () => {
    const key: ApiKey = {
      id: 7,
      name: 'restricted',
      key: '',
      status: 1,
      remain_quota: 500000,
      used_quota: 0,
      unlimited_quota: false,
      expired_time: 123,
      created_time: 1,
      accessed_time: 0,
      group: 'vip',
      auto_groups: ['vip'],
      cross_group_retry: true,
      model_limits_enabled: true,
      model_limits: 'gpt-4',
      allow_ips: '127.0.0.1',
    }
    const payload = transformFormDataToPayload(
      transformApiKeyToFormDefaults(key)
    )
    expect(payload).toEqual({
      name: 'restricted',
      remain_quota: 500000,
      unlimited_quota: false,
      group: 'default',
      auto_groups: [],
      cross_group_retry: false,
      expired_time: -1,
      model_limits_enabled: false,
      model_limits: '',
      allow_ips: '',
    })
    expect(key.group).toBe('vip')
    expect(key.expired_time).toBe(123)
  })
})
