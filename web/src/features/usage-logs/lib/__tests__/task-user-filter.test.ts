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
import { expect, test } from 'vitest'

import { buildBaseParams } from '../utils'

test('task log base params pass userId through as user_id for admin user drilldown', () => {
  const params = buildBaseParams({
    page: 2,
    pageSize: 50,
    searchParams: {
      userId: 5,
      startTime: 1788364800000,
      endTime: 1788451199000,
    },
  })

  expect(params).toEqual({
    p: 2,
    page_size: 50,
    user_id: '5',
    start_timestamp: 1788364800,
    end_timestamp: 1788451199,
  })
})
