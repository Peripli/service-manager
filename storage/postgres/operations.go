/*
 * Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package postgres

type inOp struct{}
type eqOp struct{}
type ltOp struct{}
type gtOp struct{}
type lteOp struct{}
type gteOp struct{}

func (op inOp) Get() string        { return "IN" }
func (op inOp) IsMultivalue() bool { return true }

func (op eqOp) Get() string        { return "=" }
func (op eqOp) IsMultivalue() bool { return false }

func (op ltOp) Get() string        { return "<" }
func (op ltOp) IsMultivalue() bool { return false }

func (op gtOp) Get() string        { return ">" }
func (op gtOp) IsMultivalue() bool { return false }

func (op lteOp) Get() string        { return "<=" }
func (op lteOp) IsMultivalue() bool { return false }

func (op gteOp) Get() string        { return ">=" }
func (op gteOp) IsMultivalue() bool { return false }
