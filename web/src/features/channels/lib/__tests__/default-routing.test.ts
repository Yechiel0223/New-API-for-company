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

import { channelSchema } from '../../types'
import {
  CHANNEL_FORM_DEFAULT_VALUES,
  transformFormDataToCreatePayload,
  transformFormDataToUpdatePayload,
  transformChannelToFormDefaults,
} from '../channel-form'

describe('default channel routing', () => {
  const form = {
    ...CHANNEL_FORM_DEFAULT_VALUES,
    name: 'upstream',
    type: 1,
    key: 'test-key',
    models: 'gpt-4',
    group: ['vip'],
    priority: 9,
    weight: 10,
  }
  it('creates channels with default routing regardless of stale form values', () => {
    expect(transformFormDataToCreatePayload(form).channel).toMatchObject({
      group: 'default',
      priority: 0,
      weight: 0,
      models: 'gpt-4',
    })
  })
  it('resets old routing only in the update payload', () => {
    const channel = channelSchema.parse({
      id: 7,
      name: 'upstream',
      type: 1,
      status: 1,
      key: '',
      models: 'gpt-4',
      group: 'vip',
      priority: 9,
      weight: 10,
      created_time: 1,
      test_time: 0,
      response_time: 0,
      balance: 0,
      balance_updated_time: 0,
      used_quota: 0,
    })
    const defaults = transformChannelToFormDefaults(channel)
    expect(defaults).toMatchObject({
      group: ['default'],
      priority: 0,
      weight: 0,
    })
    expect(transformFormDataToUpdatePayload(form, channel.id)).toMatchObject({
      id: 7,
      group: 'default',
      priority: 0,
      weight: 0,
    })
    expect(channel.group).toBe('vip')
  })
})
